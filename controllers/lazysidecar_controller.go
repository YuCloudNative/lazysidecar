/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"encoding/json"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"sort"

	"github.com/sirupsen/logrus"
	v1 "github.com/yucloudnative/lazysidecar/api/v1"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	"istio.io/client-go/pkg/apis/networking/v1beta1"
	versionedclient "istio.io/client-go/pkg/clientset/versioned"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	egressEnvoyFilterData              = os.Getenv("EGRESS_ENVOY_FILTER_FILE_PATH")
	workLoadEnvoyFilterData            = os.Getenv("WORKLOAD_ENVOY_FILTER_FILE_PATH")
	csmEgressImage                     = os.Getenv("CSM_EGRESS_IMAGE")
	csmEgressWorkloadPort              = os.Getenv("CSM_EGRESS_WORKLOAD_PORT")
	csmEgressWorkloadDefaultConfigData = os.Getenv("CSM_EGRESS_WORKLOAD_DEFAULT_CONFIG_DATA")
	csmEgressWorkloadNginxConfigData   = os.Getenv("CSM_EGRESS_WORKLOAD_NGINX_CONFIG_DATA")
	csmEgressWorkloadStreamConfigData  = os.Getenv("CSM_EGRESS_WORKLOAD_STREAM_CONFIG_DATA")
	ownerDeployment                    = &appv1.Deployment{}
)

// LazySidecarReconciler reconciles a LazySidecar object
type LazySidecarReconciler struct {
	client.Client
	IstioClient *versionedclient.Clientset
	Scheme      *runtime.Scheme
}

func init() {
	// if workLoadEnvoyFilterData == "" || csmEgressImage == "" || csmEgressWorkloadPort == "" || csmEgressWorkloadDefaultConfigData == "" || csmEgressWorkloadNginxConfigData == "" || csmEgressWorkloadStreamConfigData == "" {
	// 	panic("The related startup parameters are not set...")
	// }
}

//+kubebuilder:rbac:groups=yucloudnative.io,resources=lazysidecars,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=yucloudnative.io,resources=lazysidecars/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=yucloudnative.io,resources=lazysidecars/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *LazySidecarReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("reconcile LazySidecar...")
	r.Init()
	// Get current LazySidecar
	lazySidecar := &v1.LazySidecar{}
	if err := r.Get(ctx, req.NamespacedName, lazySidecar); err != nil {
		log.Error(err, "unable to fetch LazySidecar")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Reconcile CRs
	r.ReconcileSidecar(ctx, lazySidecar)
	r.ReconcileEnvoyFilter(ctx, lazySidecar)
	r.RewriteSeHostsToCsmLazySidecar(ctx, lazySidecar)

	return ctrl.Result{}, nil
}
func (r *LazySidecarReconciler) Init() {
	namespacedName := types.NamespacedName{
		Name:      v1.DefaultLazysidecarName,
		Namespace: v1.ROOTNS,
	}
	err := r.Get(context.Background(), namespacedName, ownerDeployment)
	if err != nil {
		logrus.Error("Failed to query controller: ", err)
	}
	r.ReconcileCsmEgressWorkLoad(context.Background())
}

// SetupWithManager sets up the controller with the Manager.
func (r *LazySidecarReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.LazySidecar{}).
		Owns(&v1beta1.Sidecar{}).
		Owns(&v1alpha3.EnvoyFilter{}).
		Watches(&source.Kind{Type: &appv1.Deployment{}}, &EnqueueEgressWorkLoad{csmLazySidecarReconciler: r}).
		Watches(&source.Kind{Type: &corev1.Service{}}, &EnqueueEgressWorkLoad{csmLazySidecarReconciler: r}).
		Watches(&source.Kind{Type: &corev1.ConfigMap{}}, &EnqueueEgressWorkLoad{csmLazySidecarReconciler: r}).
		Watches(&source.Kind{Type: &v1beta1.ServiceEntry{}}, &EnqueueServiceEntry{csmLazySidecarReconciler: r}).
		Complete(r)
}

func (r *LazySidecarReconciler) ReconcileCsmEgressWorkLoad(ctx context.Context) {
	r.CreateCsmSa(ctx)
	r.CreateCsmEgressConfigMap(ctx)
	r.CreateCsmEgressService(ctx)
	r.CreateCsmEgressDeployment(ctx)
}

func (r *LazySidecarReconciler) ReconcileCsmLazySidecarAddHosts(ctx context.Context, namespace string, hosts []string) {
	csmLazySidecarList := r.ListAllCsmLazySidecar(ctx)
	var handledHosts []string
	for _, host := range hosts {
		handledHosts = append(handledHosts, namespace+"/"+host)
	}
	for _, csmLazySidecar := range csmLazySidecarList.Items {
		newHosts := r.GetNewHosts(&csmLazySidecar, handledHosts)
		if len(newHosts) > 0 {
			csmLazySidecar.Spec.EgressHosts = append(csmLazySidecar.Spec.EgressHosts, newHosts...)
			err := r.Update(ctx, &csmLazySidecar)
			if err != nil {
				logrus.Error("Failed to add ServiceEntry hosts to the csmLazySidecar: ", err)
			}
		}
	}
}

func (r *LazySidecarReconciler) ReconcileCsmLazySidecarRemoveHosts(ctx context.Context, namespace string, hosts []string) {
	csmLazySidecarList := r.ListAllCsmLazySidecar(ctx)
	var handledHosts []string
	for _, host := range hosts {
		handledHosts = append(handledHosts, namespace+"/"+host)
	}
	sort.Strings(handledHosts)
	for _, csmLazySidecar := range csmLazySidecarList.Items {
		for i := 0; i < len(csmLazySidecar.Spec.EgressHosts); {
			host := csmLazySidecar.Spec.EgressHosts[i]
			if index := sort.SearchStrings(handledHosts, host); index < len(handledHosts) && handledHosts[index] == host {
				csmLazySidecar.Spec.EgressHosts = append(csmLazySidecar.Spec.EgressHosts[:i], csmLazySidecar.Spec.EgressHosts[i+1:]...)
			} else {
				i++
			}
		}
		err := r.Update(ctx, &csmLazySidecar)
		if err != nil {
			logrus.Error("Failed to remove ServiceEntry hosts to the csmLazySidecar: ", err)
		}
		r.RemoveDeriveSidecarHosts(ctx, csmLazySidecar, handledHosts)
	}
}
func (r *LazySidecarReconciler) RewriteSeHostsToCsmLazySidecar(ctx context.Context, instance *v1.LazySidecar) {
	allServiceEntryHosts := r.ListAllServiceEntryHosts(ctx)
	if instance.Spec.EgressHosts == nil {
		instance.Spec.EgressHosts = append(instance.Spec.EgressHosts, allServiceEntryHosts...)
		logrus.Debug("ServiceEntry Hosts: ", instance.Spec.EgressHosts)
		err := r.Update(ctx, instance)
		if err != nil {
			logrus.Error("Failed to update csmLazySidecar add all serviceEntry Hosts: ", err)
		}
	} else {
		newHosts := r.GetNewHosts(instance, allServiceEntryHosts)
		if len(newHosts) > 0 {
			instance.Spec.EgressHosts = append(instance.Spec.EgressHosts, newHosts...)
			err := r.Update(ctx, instance)
			if err != nil {
				logrus.Error("Failed to update csmLazySidecar add new serviceEntry Hosts: ", err)
			}
		}
	}
}

func (r *LazySidecarReconciler) ListAllServiceEntryHosts(ctx context.Context) []string {
	serviceEntryList := &v1beta1.ServiceEntryList{}
	r.List(ctx, serviceEntryList)
	var allServiceEntryHosts []string
	for _, serviceEntry := range serviceEntryList.Items {
		for _, host := range serviceEntry.Spec.Hosts {
			handledHost := serviceEntry.Namespace + "/" + host
			allServiceEntryHosts = append(allServiceEntryHosts, handledHost)
		}
	}
	return allServiceEntryHosts
}

func (r *LazySidecarReconciler) ListAllCsmLazySidecar(ctx context.Context) *v1.LazySidecarList {
	csmLazySidecarList := &v1.LazySidecarList{}
	r.List(ctx, csmLazySidecarList)
	return csmLazySidecarList
}

func (r *LazySidecarReconciler) GetNewHosts(instance *v1.LazySidecar, hosts []string) (newHosts []string) {
	sort.Strings(instance.Spec.EgressHosts)
	for _, host := range hosts {
		if index := sort.SearchStrings(instance.Spec.EgressHosts, host); index < len(instance.Spec.EgressHosts) && instance.Spec.EgressHosts[index] == host {
			continue
		} else {
			newHosts = append(newHosts, host)
		}
	}
	return
}

// GetStructDataFromString define read the Yaml configuration file and structure it
func (r *LazySidecarReconciler) ConvertJson2Struct(ctx context.Context, str string, v interface{}) error {
	err := json.Unmarshal([]byte(str), v)
	if err != nil {
		log := log.FromContext(ctx)
		log.Error(err, "convert json to struct failed")
		return err
	}
	return nil
}

func (r *LazySidecarReconciler) RemoveDeriveSidecarHosts(ctx context.Context, instance v1.LazySidecar, hosts []string) {
	sidecarInstance, err := r.IstioClient.NetworkingV1beta1().Sidecars(instance.Namespace).Get(ctx, v1.PREFIX+instance.Name, metav1.GetOptions{})
	if err != nil {
		logrus.Error("Get sidecar instance err: ", err)
		return
	}
	for i := 0; i < len(sidecarInstance.Spec.Egress[0].Hosts); {
		host := sidecarInstance.Spec.Egress[0].Hosts[i]
		if index := sort.SearchStrings(hosts, host); index < len(hosts) && hosts[index] == host {
			sidecarInstance.Spec.Egress[0].Hosts = append(sidecarInstance.Spec.Egress[0].Hosts[:i], sidecarInstance.Spec.Egress[0].Hosts[i+1:]...)
		} else {
			i++
		}
	}
	err = r.Update(ctx, sidecarInstance)
	if err != nil {
		logrus.Error("Remove sidecar hosts Failed: ", err)
	}
}
func (r *LazySidecarReconciler) GetResources() corev1.ResourceRequirements {
	resourceLimit := make(map[corev1.ResourceName]resource.Quantity)
	memLimit := resource.NewQuantity(4*2^30, resource.DecimalSI)
	resourceLimit[corev1.ResourceMemory] = *memLimit

	resourceRequest := make(map[corev1.ResourceName]resource.Quantity)
	memRequest := resource.NewQuantity(1*2^30, resource.DecimalSI)
	resourceRequest[corev1.ResourceMemory] = *memRequest

	resources := corev1.ResourceRequirements{
		Limits:   resourceLimit,
		Requests: resourceRequest,
	}
	return resources
}

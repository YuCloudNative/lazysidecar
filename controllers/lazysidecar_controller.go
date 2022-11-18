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

	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	"istio.io/client-go/pkg/apis/networking/v1beta1"
	versionedclient "istio.io/client-go/pkg/clientset/versioned"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/yucloudnative/lazysidecar/api/v1"
)

var (
// workloadEnvoyFilterPatches = os.Getenv("WORKLOAD_ENVOY_FILTER_FILE")
)

// LazySidecarReconciler reconciles a LazySidecar object
type LazySidecarReconciler struct {
	client.Client
	IstioClient *versionedclient.Clientset
	Scheme      *runtime.Scheme
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

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LazySidecarReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.LazySidecar{}).
		Owns(&v1beta1.Sidecar{}).
		Owns(&v1alpha3.EnvoyFilter{}).
		Complete(r)
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

/**
* @Author: herbguo
* @Date: 2022-3-14 15:51
 */
package controllers

import (
	"context"
	"fmt"
	"os"
	"text/template"

	networkingv1alpha3 "istio.io/api/networking/v1alpha3"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/yucloudnative/lazysidecar/api/v1"
)

func (r *LazySidecarReconciler) ReconcileEnvoyFilter(ctx context.Context, lazySidecar *v1.LazySidecar) error {
	log := log.FromContext(ctx)
	ef, err := r.IstioClient.NetworkingV1alpha3().EnvoyFilters(lazySidecar.Namespace).
		Get(ctx, v1.PREFIX+lazySidecar.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return r.createEnvoyFilter(ctx, lazySidecar)
		} else {
			return fmt.Errorf("get envoyfilter derived from LaySidecar failed. lazySidecar name : %s, namespace : %s",
				lazySidecar.Name, lazySidecar.Namespace)
		}
	}

	r.syncWorkloadSelectorToEnvoyFilter(ctx, lazySidecar, ef)
	_, err = r.IstioClient.NetworkingV1alpha3().EnvoyFilters(ef.Namespace).Update(ctx, ef, metav1.UpdateOptions{})
	if err != nil {
		log.Error(err, "fail to update sidecar by LazySidecar.",
			"namespace", lazySidecar.Namespace, "name", lazySidecar.Name)
		return err
	}
	return nil
}

func (r *LazySidecarReconciler) createEnvoyFilter(ctx context.Context, lazySidecar *v1.LazySidecar) error {
	log := log.FromContext(ctx)
	defaultEnvoyFilter, err := r.constructEnvoyFilterForLazySidecar(ctx, lazySidecar)
	if err != nil {
		log.Error(err, "fail to construct default sidecar by LazySidecar.",
			"namespace", lazySidecar.Namespace, "name", lazySidecar.Name)
		return err
	}

	_, err = r.IstioClient.NetworkingV1alpha3().EnvoyFilters(lazySidecar.Namespace).Create(ctx, defaultEnvoyFilter,
		metav1.CreateOptions{})
	if err != nil {
		log.Error(err, "fail to create default envoyfilter by LazySidecar.",
			"namespace", lazySidecar.Namespace, "name", lazySidecar.Name)
		return err
	}

	return nil
}

// overwrite WorkloadSelector in sidecar by LazySidecar.Spec.WorkloadSelector
func (r *LazySidecarReconciler) syncWorkloadSelectorToEnvoyFilter(ctx context.Context, lazySidecar *v1.LazySidecar,
	ef *v1alpha3.EnvoyFilter) {
	if lazySidecar.Spec.WorkloadSelector == nil || len(lazySidecar.Spec.WorkloadSelector) == 0 {
		// clear workload selector in sidecar
		ef.Spec.WorkloadSelector = nil
	} else {
		ef.Spec.WorkloadSelector.Labels = lazySidecar.Spec.WorkloadSelector
	}
}

func (r *LazySidecarReconciler) constructEnvoyFilterForLazySidecar(ctx context.Context,
	lazySidecar *v1.LazySidecar) (*v1alpha3.EnvoyFilter, error) {
	log := log.FromContext(ctx)
	defaultEnvoyFilterName := v1.PREFIX + lazySidecar.Name

	var workloadSelector *networkingv1alpha3.WorkloadSelector
	if lazySidecar.Spec.WorkloadSelector != nil && len(lazySidecar.Spec.WorkloadSelector) != 0 {
		// copy WorkloadSelector from LazySidecar
		workloadSelector = &networkingv1alpha3.WorkloadSelector{
			Labels: lazySidecar.Spec.WorkloadSelector,
		}
	}

	type EnvoyFilterEnvoyConfigObjectPatches struct {
		ConfigPatches []*networkingv1alpha3.EnvoyFilter_EnvoyConfigObjectPatch `json:"configPatches"`
	}

	configPatches := make([]*networkingv1alpha3.EnvoyFilter_EnvoyConfigObjectPatch, 0)
	envoyFilterEnvoyConfigObjectPatches := EnvoyFilterEnvoyConfigObjectPatches{
		ConfigPatches: configPatches,
	}

	efVars := struct {
		ServiceName            string
		Name                   string
		Namespace              string
		Port                   int
		LazysidecarGateway     string
		LazysidecarGatewayPort string
	}{
		ServiceName:            "",
		Name:                   defaultEnvoyFilterName,
		Namespace:              "",
		Port:                   0,
		LazysidecarGateway:     "",
		LazysidecarGatewayPort: "",
	}

	tpl, err := template.ParseFiles("config/envoyfilter/workload_envoyfilter_tpl.yaml")
	if err != nil {
		panic(err)
	}
	tpl.Execute(os.Stdout, &efVars)

	// data, err := os.ReadFile("config/manager/workload_envoyfilter_config_patches.json")
	// if err != nil {
		// panic(err)
	// }
	// workloadEnvoyFilterPatches := string(data)
	// log.Info("envoyfilter patches", "workloadEnvoyFilterPatches", workloadEnvoyFilterPatches)
	// r.ConvertYaml2Struct(ctx, workloadEnvoyFilterPatches, &envoyFilterEnvoyConfigObjectPatches)
	// r.ConvertJson2Struct(ctx, workloadEnvoyFilterPatches, &envoyFilterEnvoyConfigObjectPatches)

	envoyfilter := &v1alpha3.EnvoyFilter{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultEnvoyFilterName,
			Namespace: lazySidecar.Namespace,
		},
		Spec: networkingv1alpha3.EnvoyFilter{
			WorkloadSelector: workloadSelector,
			ConfigPatches:    envoyFilterEnvoyConfigObjectPatches.ConfigPatches,
		},
	}

	if err := ctrl.SetControllerReference(lazySidecar, envoyfilter, r.Scheme); err != nil {
		return nil, err
	}

	return envoyfilter, nil
}

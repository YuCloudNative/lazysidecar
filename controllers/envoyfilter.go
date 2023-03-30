/**
* @Author: herbguo
* @Date: 2022-3-14 15:51
 */
package controllers

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
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
	r.syncMiddlewarePortToEnvoyfilter(ctx, lazySidecar, ef)
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
func (r *LazySidecarReconciler) syncMiddlewarePortToEnvoyfilter(ctx context.Context, lazySidecar *v1.LazySidecar,
	ef *v1alpha3.EnvoyFilter) {

}

func (r *LazySidecarReconciler) constructEnvoyFilterForLazySidecar(ctx context.Context,
	lazySidecar *v1.LazySidecar) (*v1alpha3.EnvoyFilter, error) {
	log := log.FromContext(ctx)
	defaultEnvoyFilterName := v1.PREFIX + lazySidecar.Name

	efVars := struct {
		Services               []v1.Middleware
		Name                   string
		Namespace              string
		LazysidecarGateway     string
		LazysidecarGatewayPort string
		WorkloadSelector       map[string]string
	}{
		Services:               lazySidecar.Spec.MiddlewareList,
		Name:                   defaultEnvoyFilterName,
		Namespace:              lazySidecar.Namespace,
		LazysidecarGateway:     v1.DefaultCsmEgressServiceName,
		LazysidecarGatewayPort: "80",
		WorkloadSelector:       lazySidecar.Spec.WorkloadSelector,
	}

	tpl, err := template.ParseFiles("config/envoyfilter/workload_envoyfilter.tpl")
	if err != nil {
		log.Error(err, "Parse go template files failed.")
		return nil, err
	}
	var efBytes bytes.Buffer
	err = tpl.Execute(&efBytes, &efVars)
	if err != nil {
		log.Error(err, "go template execute failed.")
	}

	envoyfilter := &v1alpha3.EnvoyFilter{}
	err = yaml.Unmarshal(efBytes.Bytes(), envoyfilter)
	if err != nil {
		log.Error(err, "go template execute failed.")
	}

	if err := ctrl.SetControllerReference(lazySidecar, envoyfilter, r.Scheme); err != nil {
		return nil, err
	}

	return envoyfilter, nil
}

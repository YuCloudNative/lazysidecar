/**
* @Author: herbguo
* @Date: 2022-3-14 15:51
 */
package controllers

import (
	"context"
	"fmt"
	"strings"

	networkingv1beta1 "istio.io/api/networking/v1beta1"
	"istio.io/client-go/pkg/apis/networking/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/yucloudnative/lazysidecar/api/v1"
)

func (r *LazySidecarReconciler) ReconcileSidecar(ctx context.Context, lazySidecar *v1.LazySidecar) error {
	log := log.FromContext(ctx)
	sidecar, err := r.IstioClient.NetworkingV1beta1().Sidecars(lazySidecar.Namespace).
		Get(ctx, v1.PREFIX+lazySidecar.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return r.createSidecar(ctx, lazySidecar)
		} else {
			return fmt.Errorf("get sidecar derived from LaySidecar failed. lazySidecar name : %s, namespace : %s",
				lazySidecar.Name, lazySidecar.Namespace)
		}
	}

	r.syncWorkloadSelectorToSidecar(ctx, lazySidecar, sidecar)
	r.syncEgressHostsToSidecar(ctx, lazySidecar, sidecar)
	_, err = r.IstioClient.NetworkingV1beta1().Sidecars(sidecar.Namespace).Update(ctx, sidecar, metav1.UpdateOptions{})
	if err != nil {
		log.Error(err, "fail to update sidecar by LazySidecar.",
			"namespace", lazySidecar.Namespace, "name", lazySidecar.Name)
		return err
	}
	return nil
}

func (r *LazySidecarReconciler) createSidecar(ctx context.Context, lazySidecar *v1.LazySidecar) error {
	log := log.FromContext(ctx)
	defaultSidecar, err := r.constructSidecarForLazySidecar(lazySidecar)
	if err != nil {
		log.Error(err, "fail to construct default sidecar by LazySidecar.",
			"namespace", lazySidecar.Namespace, "name", lazySidecar.Name)
		return err
	}

	_, err = r.IstioClient.NetworkingV1beta1().Sidecars(lazySidecar.Namespace).Create(ctx, defaultSidecar,
		metav1.CreateOptions{})
	if err != nil {
		log.Error(err, "fail to create default sidecar by LazySidecar.",
			"namespace", lazySidecar.Namespace, "name", lazySidecar.Name)
		return err
	}

	return nil
}

// overwrite WorkloadSelector in sidecar by LazySidecar.Spec.WorkloadSelector
func (r *LazySidecarReconciler) syncWorkloadSelectorToSidecar(ctx context.Context, lazySidecar *v1.LazySidecar,
	sidecar *v1beta1.Sidecar) {
	if lazySidecar.Spec.WorkloadSelector == nil || len(lazySidecar.Spec.WorkloadSelector) == 0 {
		// clear workload selector in sidecar
		sidecar.Spec.WorkloadSelector = nil
	} else {
		sidecar.Spec.WorkloadSelector.Labels = lazySidecar.Spec.WorkloadSelector
	}
}

// overwrite egress hosts in sidecar by LazySidecar.Spec.EgressHosts
func (r *LazySidecarReconciler) syncEgressHostsToSidecar(ctx context.Context, lazySidecar *v1.LazySidecar,
	sidecar *v1beta1.Sidecar) {
	hostList := make([]string, 0)
	if lazySidecar.Spec.EgressHosts == nil || len(lazySidecar.Spec.EgressHosts) == 0 {
		// use default hosts overwriting sidecar egress hosts
		hostList = append(hostList, v1.DEFAULT_HOST)
	} else {
		for _, host := range lazySidecar.Spec.EgressHosts {
			if !strings.EqualFold(v1.DEFAULT_HOST, strings.TrimSpace(host)) {
				hostList = append(hostList, host)
			}
		}
	}

	if sidecar.Spec.Egress == nil || len(sidecar.Spec.Egress) == 0 {
		defaultEgressList := make([]*networkingv1beta1.IstioEgressListener, 0, 1)
		defaultEgress := &networkingv1beta1.IstioEgressListener{
			Hosts: hostList,
		}
		defaultEgressList = append(defaultEgressList, defaultEgress)
		sidecar.Spec.Egress = defaultEgressList
	}

	for _, egress := range sidecar.Spec.Egress {
		egress.Hosts = hostList
	}
}

func (r *LazySidecarReconciler) constructSidecarForLazySidecar(lazySidecar *v1.LazySidecar) (*v1beta1.Sidecar, error) {
	defaultSidecarName := v1.PREFIX + lazySidecar.Name

	var workloadSelector *networkingv1beta1.WorkloadSelector
	if lazySidecar.Spec.WorkloadSelector != nil && len(lazySidecar.Spec.WorkloadSelector) != 0 {
		// copy WorkloadSelector from LazySidecar
		workloadSelector = &networkingv1beta1.WorkloadSelector{
			Labels: lazySidecar.Spec.WorkloadSelector,
		}
	}

	hostList := make([]string, 0)
	hostList = append(hostList, v1.DEFAULT_HOST)
	if lazySidecar.Spec.EgressHosts != nil &&
		len(lazySidecar.Spec.EgressHosts) > 0 {
		for _, host := range lazySidecar.Spec.EgressHosts {
			if !strings.EqualFold(v1.DEFAULT_HOST, strings.TrimSpace(host)) {
				hostList = append(hostList, host)
			}
		}
	}

	// Add middleware services
	middleware := lazySidecar.Spec.MiddlewareList
	for _, m := range middleware {
		middlewareHost := fmt.Sprintf("%s/%s.%s.svc.cluster.local", m.Namespace, m.ServiceName, m.Namespace)
		hostList = append(hostList, middlewareHost)
	}

	defaultEgressList := make([]*networkingv1beta1.IstioEgressListener, 0, 1)
	defaultEgress := &networkingv1beta1.IstioEgressListener{
		Hosts: hostList,
	}
	defaultEgressList = append(defaultEgressList, defaultEgress)

	sidecar := &v1beta1.Sidecar{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultSidecarName,
			Namespace: lazySidecar.Namespace,
		},
		Spec: networkingv1beta1.Sidecar{
			WorkloadSelector: workloadSelector,
			Egress:           defaultEgressList,
		},
	}

	if err := ctrl.SetControllerReference(lazySidecar, sidecar, r.Scheme); err != nil {
		return nil, err
	}

	return sidecar, nil
}

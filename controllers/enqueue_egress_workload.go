/**
* @Author: guohb65
* @Date: 2022-3-17 16:33
 */
package controllers

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/yucloudnative/lazysidecar/api/v1"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type EnqueueEgressWorkLoad struct {
	csmLazySidecarReconciler *LazySidecarReconciler
}

func (e EnqueueEgressWorkLoad) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {}

func (e EnqueueEgressWorkLoad) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {}

func (e EnqueueEgressWorkLoad) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {}

func (e EnqueueEgressWorkLoad) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	obj1, ok1 := evt.Object.(*appv1.Deployment)
	if ok1 {
		if obj1.Name == v1.DefaultCsmEgressDeploymentName && obj1.Namespace == v1.ROOTNS {
			logrus.Info("Egress workload deployment was deleted")
			e.csmLazySidecarReconciler.CreateCsmEgressDeployment(context.Background())
		}
		if obj1.Name == v1.CsmLazySidecarBackendDeploymentName && obj1.Namespace == v1.ROOTNS {
			logrus.Info("CsmLazySidecar backend deployment was deleted")
			obj1.ResourceVersion = ""
			err := e.csmLazySidecarReconciler.Create(context.Background(), obj1)
			if err != nil {
				logrus.Error(err)
			}
		}
	}
	obj2, ok2 := evt.Object.(*corev1.Service)
	if ok2 {
		if obj2.Name == v1.DefaultCsmEgressServiceName && obj2.Namespace == v1.ROOTNS {
			logrus.Info("Egress workload service was deleted")
			e.csmLazySidecarReconciler.CreateCsmEgressService(context.Background())
		}
		if obj2.Name == v1.CsmLazySidecarBackendServiceName && obj2.Namespace == v1.ROOTNS {
			logrus.Info("CsmLazySidecar backend service was deleted")
			obj2.ResourceVersion = ""
			err := e.csmLazySidecarReconciler.Create(context.Background(), obj2)
			if err != nil {
				logrus.Error(err)
			}
		}
	}
	obj3, ok3 := evt.Object.(*corev1.ConfigMap)
	if ok3 {
		if obj3.Name == v1.DefaultCsmEgressConfigmapName && obj3.Namespace == v1.ROOTNS {
			logrus.Info("Egress workload configmap was deleted")
			e.csmLazySidecarReconciler.CreateCsmEgressConfigMap(context.Background())
		}
	}
}

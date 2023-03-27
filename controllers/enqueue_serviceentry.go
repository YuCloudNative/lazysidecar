/**
* @Author: guohb65
* @Date: 2022-3-17 16:33
 */
package controllers

import (
	"context"
	"github.com/sirupsen/logrus"
	"istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type EnqueueServiceEntry struct {
	csmLazySidecarReconciler *LazySidecarReconciler
}

func (e EnqueueServiceEntry) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	logrus.Debug("ServiceEntry create event")
	obj, ok := evt.Object.(*v1beta1.ServiceEntry)
	if ok {
		ns := obj.Namespace
		hosts := obj.Spec.Hosts
		e.csmLazySidecarReconciler.ReconcileCsmLazySidecarAddHosts(context.Background(), ns, hosts)
	}
}

func (e EnqueueServiceEntry) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	logrus.Debug("ServiceEntry Update event")

	oldObj, ok1 := evt.ObjectOld.(*v1beta1.ServiceEntry)
	newObj, ok2 := evt.ObjectNew.(*v1beta1.ServiceEntry)
	if ok1 && ok2 {
		oldNs := oldObj.Namespace
		newNs := newObj.Namespace
		oldObjhosts := oldObj.Spec.Hosts
		newObjhosts := newObj.Spec.Hosts
		e.csmLazySidecarReconciler.ReconcileCsmLazySidecarRemoveHosts(context.Background(), oldNs, oldObjhosts)
		e.csmLazySidecarReconciler.ReconcileCsmLazySidecarAddHosts(context.Background(), newNs, newObjhosts)
	}
}

func (e EnqueueServiceEntry) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
}

func (e EnqueueServiceEntry) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	logrus.Debug("ServiceEntry Delete event")

	obj, ok := evt.Object.(*v1beta1.ServiceEntry)
	if ok {
		ns := obj.Namespace
		hosts := obj.Spec.Hosts
		e.csmLazySidecarReconciler.ReconcileCsmLazySidecarRemoveHosts(context.Background(), ns, hosts)
	}
}

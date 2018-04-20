package controller

import (
	"reflect"

	kapiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	kubervisorapi "github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
	"github.com/amadeusitgroup/podkubervisor/pkg/labeling"
)

func (ctrl *Controller) onAddKubervisorService(obj interface{}) {
	bc, ok := obj.(*kubervisorapi.KubervisorService)
	if !ok {
		ctrl.Logger.Sugar().Errorf("adding KubervisorService, expected KubervisorService object. Got: %+v", obj)
		return
	}
	ctrl.Logger.Sugar().Debugf("onAddKubervisorService %s/%s", bc.Namespace, bc.Name)
	if !reflect.DeepEqual(bc.Status, kubervisorapi.KubervisorServiceStatus{}) {
		ctrl.Logger.Sugar().Errorf("KubervisorService %s/%s created with non empty status. Going to be removed", bc.Namespace, bc.Name)

		if _, err := cache.MetaNamespaceKeyFunc(bc); err != nil {
			ctrl.Logger.Sugar().Errorf("couldn't get key for KubervisorService (to be deleted) %s/%s: %v", bc.Namespace, bc.Name, err)
			return
		}
		// TODO: how to remove a kubervisorservice created with an invalid or even with a valid status. What in case of error for this delete?

		return
	}

	ctrl.enqueueFunc(bc)
}

func (ctrl *Controller) onDeleteKubervisorService(obj interface{}) {
	_, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		ctrl.Logger.Sugar().Errorf("Unable to get key for %#v: %v", obj, err)
		return
	}
	bc, ok := obj.(*kubervisorapi.KubervisorService)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			ctrl.Logger.Sugar().Errorf("unknown object from KubervisorService delete event: %#v", obj)
			return
		}
		bc, ok = tombstone.Obj.(*kubervisorapi.KubervisorService)
		if !ok {
			ctrl.Logger.Sugar().Errorf("Tombstone contained object that is not an KubervisorService: %#v", obj)
			return
		}
	}
	ctrl.Logger.Sugar().Debugf("onDeleteKubervisorService %s/%s", bc.Namespace, bc.Name)

	ctrl.enqueueFunc(bc)
}

func (ctrl *Controller) onUpdateKubervisorService(oldObj, newObj interface{}) {
	bc, ok := newObj.(*kubervisorapi.KubervisorService)
	if !ok {
		ctrl.Logger.Sugar().Errorf("updating KubervisorService, expected KubervisorService object. Got: %+v", newObj)
		return
	}
	ctrl.Logger.Sugar().Debugf("onUpdateKubervisorService %s/%s", bc.Namespace, bc.Name)

	ctrl.enqueueFunc(bc)
}

func (ctrl *Controller) onAddPod(obj interface{}) {
	pod, ok := obj.(*kapiv1.Pod)
	if !ok {
		ctrl.Logger.Sugar().Errorf("adding Pod, expected Pod object. Got: %+v", obj)
		return
	}
	ctrl.Logger.Sugar().Debugf("onAddPod %s/%s", pod.Namespace, pod.Name)
	ctrl.podAction(pod)
}

func (ctrl *Controller) onDeletePod(obj interface{}) {
	_, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		ctrl.Logger.Sugar().Errorf("Unable to get key for %#v: %v", obj, err)
		return
	}
	pod, ok := obj.(*kapiv1.Pod)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			ctrl.Logger.Sugar().Errorf("unknown object from Pod delete event: %#v", obj)
			return
		}
		pod, ok = tombstone.Obj.(*kapiv1.Pod)
		if !ok {
			ctrl.Logger.Sugar().Errorf("Tombstone contained object that is not an Pod: %#v", obj)
			return
		}
	}
	ctrl.Logger.Sugar().Debugf("onDeletePod %s/%s", pod.Namespace, pod.Name)
	if ksName, ok := pod.Labels[labeling.LabelBreakerNameKey]; ok {
		// Pod already handled by a KubervisorService
		ks, err := ctrl.breakerLister.KubervisorServices(pod.Namespace).Get(ksName)
		if err != nil {
			ctrl.Logger.Sugar().Errorf("unable to get KubervisorService:%s, associated in pod:%s/%s err: %v", ksName, pod.Namespace, pod.Name, err)
			return
		}
		ctrl.enqueueFunc(ks)
		return
	}

}

func (ctrl *Controller) onUpdatePod(oldObj, newObj interface{}) {
	pod, ok := newObj.(*kapiv1.Pod)
	if !ok {
		ctrl.Logger.Sugar().Errorf("update Pod, expected Pod object. Got: %+v", newObj)
		return
	}
	ctrl.Logger.Sugar().Debugf("onUpdatePod %s/%s", pod.Namespace, pod.Name)
	ctrl.podAction(pod)
}

func (ctrl *Controller) podAction(pod *kapiv1.Pod) {
	if ksName, ok := pod.Labels[labeling.LabelBreakerNameKey]; ok {
		// Pod already handle by a KubervisorService
		ks, err := ctrl.breakerLister.KubervisorServices(pod.Namespace).Get(ksName)
		if err != nil {
			ctrl.Logger.Sugar().Errorf("unable to get KubervisorService:%s, associated in pod:%s/%s err: %v", ksName, pod.Namespace, pod.Name, err)
			return
		}
		ctrl.enqueueFunc(ks)
		return
	}

	kss, err := ctrl.breakerLister.List(labels.Everything())
	if err != nil {
		ctrl.Logger.Sugar().Errorf("unable to list KubervisorService, err: %v", err)
		return
	}
	for _, ks := range kss {
		if ks.Namespace != pod.Namespace {
			continue
		}
		svcName := ks.Spec.Service
		svc, err := ctrl.serviceLister.Services(ks.Namespace).Get(svcName)
		if err != nil {
			ctrl.Logger.Sugar().Errorf("unable to get Service: %s/%s, err: %v", ks.Namespace, svcName, err)
			return
		}
		podSelector := svc.DeepCopy().Spec.Selector
		// remove LabelTraffic if it is already present
		delete(podSelector, labeling.LabelTrafficKey)
		delete(podSelector, labeling.LabelBreakerNameKey)
		selector := labels.SelectorFromSet(labels.Set(podSelector))
		if selector.Matches(labels.Set(pod.Labels)) {
			ctrl.enqueueFunc(ks)
			return
		}
	}
}

func (ctrl *Controller) onAddService(obj interface{}) {
	svc, ok := obj.(*kapiv1.Service)
	if !ok {
		ctrl.Logger.Sugar().Errorf("adding Service, expected Service object. Got: %+v", obj)
		return
	}
	ctrl.Logger.Sugar().Debugf("onAddService %s/%s", svc.Namespace, svc.Name)
	ctrl.serviceAction(svc)
}

func (ctrl *Controller) onDeleteService(obj interface{}) {
	_, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		ctrl.Logger.Sugar().Errorf("Unable to get key for %#v: %v", obj, err)
		return
	}
	svc, ok := obj.(*kapiv1.Service)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			ctrl.Logger.Sugar().Errorf("unknown object from Service delete event: %#v", obj)
			return
		}
		svc, ok = tombstone.Obj.(*kapiv1.Service)
		if !ok {
			ctrl.Logger.Sugar().Errorf("Tombstone contained object that is not an Service: %#v", obj)
			return
		}
	}
	ctrl.Logger.Sugar().Debugf("onDeletePod %s/%s", svc.Namespace, svc.Name)
	ctrl.serviceAction(svc)
}

func (ctrl *Controller) onUpdateService(oldObj, newObj interface{}) {
	svc, ok := newObj.(*kapiv1.Service)
	if !ok {
		ctrl.Logger.Sugar().Errorf("update Service, expected Service object. Got: %+v", newObj)
		return
	}
	ctrl.Logger.Sugar().Debugf("onUpdateService %s/%s", svc.Namespace, svc.Name)
	ctrl.serviceAction(svc)
}

func (ctrl *Controller) serviceAction(svc *kapiv1.Service) {
	kss, err := ctrl.breakerLister.List(labels.Everything())
	if err != nil {
		ctrl.Logger.Sugar().Errorf("unable to list KubervisorService, err: %v", err)
		return
	}
	for _, ks := range kss {
		if svc.Namespace == ks.Namespace && svc.Name == ks.Spec.Service {
			ctrl.enqueueFunc(ks)
			return
		}
	}
}

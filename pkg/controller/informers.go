package controller

import (
	"reflect"

	"k8s.io/client-go/tools/cache"

	kubervisorapi "github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
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
}

func (ctrl *Controller) onDeletePod(obj interface{}) {
}

func (ctrl *Controller) onUpdatePod(oldObj, newObj interface{}) {
}

func (ctrl *Controller) onAddService(obj interface{}) {
}

func (ctrl *Controller) onDeleteService(obj interface{}) {
}

func (ctrl *Controller) onUpdateService(oldObj, newObj interface{}) {
}

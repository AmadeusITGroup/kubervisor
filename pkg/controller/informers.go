package controller

import (
	"reflect"

	"k8s.io/client-go/tools/cache"

	kubervisorapi "github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
)

func (ctrl *Controller) onAddBreakerConfig(obj interface{}) {
	bc, ok := obj.(*kubervisorapi.BreakerConfig)
	if !ok {
		ctrl.Logger.Sugar().Errorf("adding BreakerConfig, expected BreakerConfig object. Got: %+v", obj)
		return
	}
	ctrl.Logger.Sugar().Debugf("onAddBreakerConfig %s/%s", bc.Namespace, bc.Name)
	if !reflect.DeepEqual(bc.Status, kubervisorapi.BreakerConfigStatus{}) {
		ctrl.Logger.Sugar().Errorf("BreakerConfig %s/%s created with non empty status. Going to be removed", bc.Namespace, bc.Name)

		if _, err := cache.MetaNamespaceKeyFunc(bc); err != nil {
			ctrl.Logger.Sugar().Errorf("couldn't get key for BreakerConfig (to be deleted) %s/%s: %v", bc.Namespace, bc.Name, err)
			return
		}
		// TODO: how to remove a breakerconfig created with an invalid or even with a valid status. What in case of error for this delete?
		if err := ctrl.deleteBreakerConfig(bc.Namespace, bc.Name); err != nil {
			ctrl.Logger.Sugar().Errorf("unable to delete non empty status BreakerConfig %s/%s: %v. No retry will be performed.", bc.Namespace, bc.Name, err)
		}

		return
	}

	ctrl.enqueueFunc(bc)
}

func (ctrl *Controller) onDeleteBreakerConfig(obj interface{}) {
	_, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		ctrl.Logger.Sugar().Errorf("Unable to get key for %#v: %v", obj, err)
		return
	}
	bc, ok := obj.(*kubervisorapi.BreakerConfig)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			ctrl.Logger.Sugar().Errorf("unknown object from BreakerConfig delete event: %#v", obj)
			return
		}
		bc, ok = tombstone.Obj.(*kubervisorapi.BreakerConfig)
		if !ok {
			ctrl.Logger.Sugar().Errorf("Tombstone contained object that is not an BreakerConfig: %#v", obj)
			return
		}
	}
	ctrl.Logger.Sugar().Debugf("onDeleteBreakerConfig %s/%s", bc.Namespace, bc.Name)

	ctrl.enqueueFunc(bc)
}

func (ctrl *Controller) onUpdateBreakerConfig(oldObj, newObj interface{}) {
	bc, ok := newObj.(*kubervisorapi.BreakerConfig)
	if !ok {
		ctrl.Logger.Sugar().Errorf("updating BreakerConfig, expected BreakerConfig object. Got: %+v", newObj)
		return
	}
	ctrl.Logger.Sugar().Debugf("onUpdateBreakerConfig %s/%s", bc.Namespace, bc.Name)

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

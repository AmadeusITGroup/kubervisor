package controller

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	kubervisorapi "github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
	"github.com/amadeusitgroup/podkubervisor/pkg/controller/item"
)

// IsSpecUpdated return true if the the BreakerConfig have been updated
func IsSpecUpdated(bc *kubervisorapi.BreakerConfig, svc *corev1.Service, bci item.Interface) bool {
	selector := labels.Set(svc.Spec.Selector).AsSelectorPreValidated()
	return bci.CompareWithSpec(&bc.Spec, selector)
}

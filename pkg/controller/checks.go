package controller

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	kubervisorapi "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1"
	"github.com/amadeusitgroup/kubervisor/pkg/controller/item"
)

// IsSpecUpdated return true if the the KubervisorService have been updated
func IsSpecUpdated(bc *kubervisorapi.KubervisorService, svc *corev1.Service, bci item.Interface) bool {
	selector := labels.Set(svc.Spec.Selector).AsSelectorPreValidated()
	return bci.CompareWithSpec(&bc.Spec, selector)
}

package controller

import (
	kapiv1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1alpha1"
)

type statusUpdateFunc func(*api.KubervisorServiceStatus, string, metav1.Time) (*api.KubervisorServiceStatus, error)

// newStatusCondition used to create a new newStatusCondition
func newStatusCondition(conditionType api.KubervisorServiceConditionType, status kapiv1.ConditionStatus, msg, reason string, creationTime metav1.Time) api.KubervisorServiceCondition {
	return api.KubervisorServiceCondition{
		Type:          conditionType,
		Status:        status,
		LastProbeTime: creationTime,
		Reason:        reason,
		Message:       msg,
	}
}

// newStatusConditionServiceError used to create a new KubervisorServiceCondition for service not found
func newStatusConditionServiceError(msg string, creationTime metav1.Time) api.KubervisorServiceCondition {
	return newStatusCondition(api.KubeServiceNotAvailable, kapiv1.ConditionTrue, msg, "Kubernetes service not found", creationTime)
}

// updateStatusCondition used to create a new KubervisorServiceCondition for service not found
func updateStatusCondition(old *api.KubervisorServiceCondition, status kapiv1.ConditionStatus, updateTime metav1.Time) api.KubervisorServiceCondition {
	newCondition := old.DeepCopy()
	{
		newCondition.LastProbeTime = updateTime
		newCondition.Status = status
	}
	return *newCondition
}

// UpdateStatusConditionServiceError used to udpate or create a KubervisorServiceCondition for Kubernetes service not found
func UpdateStatusConditionServiceError(status *api.KubervisorServiceStatus, msg string, updatetime metav1.Time) (*api.KubervisorServiceStatus, error) {
	newFunc := func() api.KubervisorServiceCondition {
		return newStatusConditionServiceError(msg, updatetime)
	}
	upFunc := func(old *api.KubervisorServiceCondition) api.KubervisorServiceCondition {
		return updateStatusCondition(old, kapiv1.ConditionTrue, updatetime)
	}

	return UpdateStatusCondition(status, api.KubeServiceNotAvailable, updatetime, newFunc, upFunc)
}

// newStatusConditionInitFailed used to create a new KubervisorServiceCondition for initialization failure
func newStatusConditionInitFailed(msg string, creationTime metav1.Time) api.KubervisorServiceCondition {
	return newStatusCondition(api.KubervisorServiceInitFailed, kapiv1.ConditionTrue, msg, "KubervisorService initialization failed", creationTime)
}

// UpdateStatusConditionInitFailure used to udpate or create a KubervisorServiceCondition for Initialization failure
func UpdateStatusConditionInitFailure(status *api.KubervisorServiceStatus, msg string, updatetime metav1.Time) (*api.KubervisorServiceStatus, error) {
	newFunc := func() api.KubervisorServiceCondition {
		return newStatusConditionInitFailed(msg, updatetime)
	}
	upFunc := func(old *api.KubervisorServiceCondition) api.KubervisorServiceCondition {
		return updateStatusCondition(old, kapiv1.ConditionTrue, updatetime)
	}

	return UpdateStatusCondition(status, api.KubervisorServiceInitFailed, updatetime, newFunc, upFunc)
}

// newStatusConditionRunning used to create a new KubervisorServiceCondition for running
func newStatusConditionRunning(msg string, creationTime metav1.Time) api.KubervisorServiceCondition {
	return newStatusCondition(api.KubervisorServiceRunning, kapiv1.ConditionTrue, msg, "KubervisorService running", creationTime)
}

// UpdateStatusConditionRunning used to udpate or create a KubervisorServiceCondition for Running
func UpdateStatusConditionRunning(status *api.KubervisorServiceStatus, msg string, updatetime metav1.Time) (*api.KubervisorServiceStatus, error) {
	newFunc := func() api.KubervisorServiceCondition {
		return newStatusConditionRunning(msg, updatetime)
	}
	upFunc := func(old *api.KubervisorServiceCondition) api.KubervisorServiceCondition {
		return updateStatusCondition(old, kapiv1.ConditionTrue, updatetime)
	}

	return UpdateStatusCondition(status, api.KubervisorServiceRunning, updatetime, newFunc, upFunc)
}

// UpdateStatusCondition used to udpate or create a KubervisorServiceCondition
func UpdateStatusCondition(status *api.KubervisorServiceStatus, conditionType api.KubervisorServiceConditionType, updatetime metav1.Time, newConditionFunc func() api.KubervisorServiceCondition, updateConditionFunc func(old *api.KubervisorServiceCondition) api.KubervisorServiceCondition) (*api.KubervisorServiceStatus, error) {
	newStatus := status.DeepCopy()
	found := false
	for idCondition, condition := range newStatus.Conditions {
		if condition.Type == conditionType {
			found = true
			newStatus.Conditions[idCondition] = updateConditionFunc(&condition)
		} else {
			// TODO improve condition status transition. Can we have 2 condition with true ?
			newStatus.Conditions[idCondition] = updateStatusCondition(&condition, kapiv1.ConditionFalse, updatetime)
		}
	}
	if !found {
		newStatus.Conditions = append(newStatus.Conditions, newConditionFunc())
	}

	return newStatus, nil
}

func equalPodCountStatus(a, b api.PodCountStatus) bool {
	t0 := metav1.Time{}
	a.LastProbeTime, b.LastProbeTime = t0, t0
	return apiequality.Semantic.DeepEqual(a, b)
}

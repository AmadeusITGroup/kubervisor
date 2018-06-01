package controller

import (
	kapiv1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv1 "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1"
)

type statusUpdateFunc func(*apiv1.KubervisorServiceStatus, string, metav1.Time) (*apiv1.KubervisorServiceStatus, error)

// newStatusCondition used to create a new newStatusCondition
func newStatusCondition(conditionType apiv1.KubervisorServiceConditionType, status kapiv1.ConditionStatus, msg, reason string, creationTime metav1.Time) apiv1.KubervisorServiceCondition {
	return apiv1.KubervisorServiceCondition{
		Type:          conditionType,
		Status:        status,
		LastProbeTime: creationTime,
		Reason:        reason,
		Message:       msg,
	}
}

// newStatusConditionServiceError used to create a new KubervisorServiceCondition for service not found
func newStatusConditionServiceError(msg string, creationTime metav1.Time) apiv1.KubervisorServiceCondition {
	return newStatusCondition(apiv1.KubeServiceNotAvailable, kapiv1.ConditionTrue, msg, "Kubernetes service not found", creationTime)
}

// updateStatusCondition used to create a new KubervisorServiceCondition for service not found
func updateStatusCondition(old *apiv1.KubervisorServiceCondition, status kapiv1.ConditionStatus, updateTime metav1.Time) apiv1.KubervisorServiceCondition {
	newCondition := old.DeepCopy()
	{
		newCondition.LastProbeTime = updateTime
		newCondition.Status = status
	}
	return *newCondition
}

// UpdateStatusConditionServiceError used to udpate or create a KubervisorServiceCondition for Kubernetes service not found
func UpdateStatusConditionServiceError(status *apiv1.KubervisorServiceStatus, msg string, updatetime metav1.Time) (*apiv1.KubervisorServiceStatus, error) {
	newFunc := func() apiv1.KubervisorServiceCondition {
		return newStatusConditionServiceError(msg, updatetime)
	}
	upFunc := func(old *apiv1.KubervisorServiceCondition) apiv1.KubervisorServiceCondition {
		return updateStatusCondition(old, kapiv1.ConditionTrue, updatetime)
	}

	return UpdateStatusCondition(status, apiv1.KubeServiceNotAvailable, updatetime, newFunc, upFunc)
}

// newStatusConditionInitFailed used to create a new KubervisorServiceCondition for initialization failure
func newStatusConditionInitFailed(msg string, creationTime metav1.Time) apiv1.KubervisorServiceCondition {
	return newStatusCondition(apiv1.KubervisorServiceInitFailed, kapiv1.ConditionTrue, msg, "KubervisorService initialization failed", creationTime)
}

// UpdateStatusConditionInitFailure used to udpate or create a KubervisorServiceCondition for Initialization failure
func UpdateStatusConditionInitFailure(status *apiv1.KubervisorServiceStatus, msg string, updatetime metav1.Time) (*apiv1.KubervisorServiceStatus, error) {
	newFunc := func() apiv1.KubervisorServiceCondition {
		return newStatusConditionInitFailed(msg, updatetime)
	}
	upFunc := func(old *apiv1.KubervisorServiceCondition) apiv1.KubervisorServiceCondition {
		return updateStatusCondition(old, kapiv1.ConditionTrue, updatetime)
	}

	return UpdateStatusCondition(status, apiv1.KubervisorServiceInitFailed, updatetime, newFunc, upFunc)
}

// newStatusConditionRunning used to create a new KubervisorServiceCondition for running
func newStatusConditionRunning(msg string, creationTime metav1.Time) apiv1.KubervisorServiceCondition {
	return newStatusCondition(apiv1.KubervisorServiceRunning, kapiv1.ConditionTrue, msg, "KubervisorService running", creationTime)
}

// UpdateStatusConditionRunning used to udpate or create a KubervisorServiceCondition for Running
func UpdateStatusConditionRunning(status *apiv1.KubervisorServiceStatus, msg string, updatetime metav1.Time) (*apiv1.KubervisorServiceStatus, error) {
	newFunc := func() apiv1.KubervisorServiceCondition {
		return newStatusConditionRunning(msg, updatetime)
	}
	upFunc := func(old *apiv1.KubervisorServiceCondition) apiv1.KubervisorServiceCondition {
		return updateStatusCondition(old, kapiv1.ConditionTrue, updatetime)
	}

	return UpdateStatusCondition(status, apiv1.KubervisorServiceRunning, updatetime, newFunc, upFunc)
}

// UpdateStatusCondition used to udpate or create a KubervisorServiceCondition
func UpdateStatusCondition(status *apiv1.KubervisorServiceStatus, conditionType apiv1.KubervisorServiceConditionType, updatetime metav1.Time, newConditionFunc func() apiv1.KubervisorServiceCondition, updateConditionFunc func(old *apiv1.KubervisorServiceCondition) apiv1.KubervisorServiceCondition) (*apiv1.KubervisorServiceStatus, error) {
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

func equalPodCountStatus(a, b apiv1.PodCountStatus) bool {
	t0 := metav1.Time{}
	a.LastProbeTime, b.LastProbeTime = t0, t0
	return apiequality.Semantic.DeepEqual(a, b)
}

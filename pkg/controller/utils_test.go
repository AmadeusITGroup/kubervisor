package controller

import (
	"reflect"
	"testing"
	"time"

	apiv1 "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1alpha1"
	kapiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUpdateStatusConditionServiceError(t *testing.T) {
	now := metav1.Now()
	pastTime := metav1.NewTime(now.Truncate(2 * time.Minute))
	msg := "service not found"

	type args struct {
		status     *apiv1.KubervisorServiceStatus
		msg        string
		updatetime metav1.Time
	}
	tests := []struct {
		name    string
		args    args
		want    *apiv1.KubervisorServiceStatus
		wantErr bool
	}{
		{
			name: "add new condition",
			args: args{
				status: &apiv1.KubervisorServiceStatus{
					Conditions: []apiv1.KubervisorServiceCondition{},
				},
				msg:        msg,
				updatetime: now,
			},
			want: &apiv1.KubervisorServiceStatus{
				Conditions: []apiv1.KubervisorServiceCondition{newStatusConditionServiceError(msg, now)},
			},
			wantErr: false,
		},

		{
			name: "update condition",
			args: args{
				status: &apiv1.KubervisorServiceStatus{
					Conditions: []apiv1.KubervisorServiceCondition{newStatusConditionServiceError(msg, pastTime)},
				},
				msg:        msg,
				updatetime: now,
			},
			want: &apiv1.KubervisorServiceStatus{
				Conditions: []apiv1.KubervisorServiceCondition{newStatusConditionServiceError(msg, now)},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UpdateStatusConditionServiceError(tt.args.status, tt.args.msg, tt.args.updatetime)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateStatusConditionServiceError() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateStatusConditionServiceError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateStatusConditionInitFailure(t *testing.T) {
	now := metav1.Now()
	pastTime := metav1.NewTime(now.Truncate(2 * time.Minute))
	msg := "bad breaker config"

	type args struct {
		status     *apiv1.KubervisorServiceStatus
		msg        string
		updatetime metav1.Time
	}
	tests := []struct {
		name    string
		args    args
		want    *apiv1.KubervisorServiceStatus
		wantErr bool
	}{
		{
			name: "add new condition",
			args: args{
				status: &apiv1.KubervisorServiceStatus{
					Conditions: []apiv1.KubervisorServiceCondition{},
				},
				msg:        msg,
				updatetime: now,
			},
			want: &apiv1.KubervisorServiceStatus{
				Conditions: []apiv1.KubervisorServiceCondition{newStatusConditionInitFailed(msg, now)},
			},
			wantErr: false,
		},

		{
			name: "update condition",
			args: args{
				status: &apiv1.KubervisorServiceStatus{
					Conditions: []apiv1.KubervisorServiceCondition{newStatusConditionInitFailed(msg, pastTime)},
				},
				msg:        msg,
				updatetime: now,
			},
			want: &apiv1.KubervisorServiceStatus{
				Conditions: []apiv1.KubervisorServiceCondition{newStatusConditionInitFailed(msg, now)},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UpdateStatusConditionInitFailure(tt.args.status, tt.args.msg, tt.args.updatetime)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateStatusConditionInitFailure() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateStatusConditionInitFailure() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateStatusConditionRunning(t *testing.T) {

	now := metav1.Now()
	pastTime := metav1.NewTime(now.Truncate(2 * time.Minute))
	msg := "running breaker config"

	initFailedCondition := newStatusConditionInitFailed("init has failed", pastTime)
	initFailedConditionUpdated := updateStatusCondition(&initFailedCondition, kapiv1.ConditionFalse, now)

	type args struct {
		status     *apiv1.KubervisorServiceStatus
		msg        string
		updatetime metav1.Time
	}
	tests := []struct {
		name    string
		args    args
		want    *apiv1.KubervisorServiceStatus
		wantErr bool
	}{
		{
			name: "add new condition",
			args: args{
				status: &apiv1.KubervisorServiceStatus{
					Conditions: []apiv1.KubervisorServiceCondition{
						initFailedCondition,
					},
				},
				msg:        msg,
				updatetime: now,
			},
			want: &apiv1.KubervisorServiceStatus{
				Conditions: []apiv1.KubervisorServiceCondition{
					initFailedConditionUpdated,
					newStatusConditionRunning(msg, now),
				},
			},
			wantErr: false,
		},
		{
			name: "update condition",
			args: args{
				status: &apiv1.KubervisorServiceStatus{
					Conditions: []apiv1.KubervisorServiceCondition{newStatusConditionRunning(msg, pastTime)},
				},
				msg:        msg,
				updatetime: now,
			},
			want: &apiv1.KubervisorServiceStatus{
				Conditions: []apiv1.KubervisorServiceCondition{newStatusConditionRunning(msg, now)},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UpdateStatusConditionRunning(tt.args.status, tt.args.msg, tt.args.updatetime)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateStatusConditionRunning() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateStatusConditionRunning()\ngot = %v\nwant= %v\n", got, tt.want)
			}
		})
	}
}

func Test_equalPodCountStatus(t *testing.T) {
	t0 := metav1.Time{}
	t1 := metav1.Time{Time: time.Now()}

	type args struct {
		a apiv1.PodCountStatus
		b apiv1.PodCountStatus
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "equal",
			args: args{
				a: apiv1.PodCountStatus{
					LastProbeTime: t0,
					NbPodsBreaked: 1,
					NbPodsManaged: 10,
				},
				b: apiv1.PodCountStatus{
					LastProbeTime: t1,
					NbPodsBreaked: 1,
					NbPodsManaged: 10,
				},
			},
			want: true,
		},
		{
			name: "diff",
			args: args{
				a: apiv1.PodCountStatus{
					LastProbeTime: t0,
					NbPodsBreaked: 1,
					NbPodsManaged: 10,
				},
				b: apiv1.PodCountStatus{
					LastProbeTime: t1,
					NbPodsBreaked: 0,
					NbPodsManaged: 10,
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := equalPodCountStatus(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("equalPodCountStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

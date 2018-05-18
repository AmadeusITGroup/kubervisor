package controller

import (
	"context"
	"testing"

	kubervisorapi "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1alpha1"
	"github.com/amadeusitgroup/kubervisor/pkg/controller/item"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type testInterface struct {
	name                string
	namespace           string
	StartFunc           func(ctx context.Context)
	StopFunc            func() error
	CompareWithSpecFunc func(spec *kubervisorapi.KubervisorServiceSpec, selector labels.Selector) bool
	GetStatusFunc       func() kubervisorapi.PodCountStatus
}

func (ei *testInterface) Name() string {
	return ei.name
}
func (ei *testInterface) Namespace() string {
	return ei.namespace
}
func (ei *testInterface) Start(ctx context.Context) {
	if ei.StartFunc != nil {
		ei.StartFunc(ctx)
	}
	return
}
func (ei *testInterface) Stop() error {
	if ei.StopFunc != nil {
		return ei.StopFunc()
	}
	return nil
}
func (ei *testInterface) CompareWithSpec(spec *kubervisorapi.KubervisorServiceSpec, selector labels.Selector) bool {
	if ei.CompareWithSpecFunc != nil {
		return ei.CompareWithSpecFunc(spec, selector)
	}
	return true
}
func (ei *testInterface) GetStatus() kubervisorapi.PodCountStatus {
	if ei.GetStatusFunc != nil {
		return ei.GetStatusFunc()
	}
	return kubervisorapi.PodCountStatus{}
}

func TestIsSpecUpdated(t *testing.T) {
	type args struct {
		bc  *kubervisorapi.KubervisorService
		svc *corev1.Service
		bci item.Interface
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "ok",
			args: args{
				bc: &kubervisorapi.KubervisorService{
					ObjectMeta: metav1.ObjectMeta{Name: "test-bc", Namespace: "test-ns"},
					Spec:       kubervisorapi.KubervisorServiceSpec{},
				},
				svc: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{Name: "test-svc", Namespace: "test-ns"},
					Spec: corev1.ServiceSpec{
						Selector: map[string]string{"app": "foo"},
					},
				},
				bci: &testInterface{
					CompareWithSpecFunc: func(spec *kubervisorapi.KubervisorServiceSpec, selector labels.Selector) bool { return true },
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSpecUpdated(tt.args.bc, tt.args.svc, tt.args.bci); got != tt.want {
				t.Errorf("IsSpecUpdated() = %v, want %v", got, tt.want)
			}
		})
	}
}

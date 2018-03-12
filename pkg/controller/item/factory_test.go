package item

import (
	"testing"

	apiv1 "github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
	test "github.com/amadeusitgroup/podkubervisor/test"
	kapiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func TestNew(t *testing.T) {
	activatorStrategyConfig := apiv1.DefaultActivatorStrategy(&apiv1.ActivatorStrategy{})
	breakerStrategyConfig := apiv1.DefaultBreakerStrategy(&apiv1.BreakerStrategy{
		DiscreteValueOutOfList: &apiv1.DiscreteValueOutOfList{
			PromQL:            "query",
			PrometheusService: "Service",
			GoodValues:        []string{"ok"},
			Key:               "code",
			PodNameKey:        "podname",
		},
	})

	type args struct {
		bc  *apiv1.KubervisorService
		cfg *Config
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "create simple KubervisorServiceItem",
			args: args{
				bc: &apiv1.KubervisorService{
					ObjectMeta: metav1.ObjectMeta{Name: "test-bc", Namespace: "test-ns"},
					Spec: apiv1.KubervisorServiceSpec{
						Activator: *activatorStrategyConfig,
						Breaker:   *breakerStrategyConfig,
					},
				},
				cfg: &Config{
					PodLister: test.NewTestPodLister([]*kapiv1.Pod{}),
					Selector:  labels.Set(map[string]string{"app": "foo"}).AsSelector(),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.args.bc, tt.args.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

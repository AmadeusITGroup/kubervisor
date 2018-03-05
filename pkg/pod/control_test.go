package pod

import (
	"testing"

	kapiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	kfakeclient "k8s.io/client-go/kubernetes/fake"

	"github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
	"github.com/amadeusitgroup/podkubervisor/pkg/labeling"

	test "github.com/amadeusitgroup/podkubervisor/test"
)

func TestControl_UpdateBreakerAnnotationAndLabel(t *testing.T) {

	checkFunc1 := func(t *testing.T, p *kapiv1.Pod) bool {
		if traffic, ok := p.Labels[labeling.LabelTrafficKey]; traffic != string(labeling.LabelTrafficNo) || !ok {
			t.Errorf("bad traffic label")
			return false
		}
		if name, ok := p.Labels[labeling.LabelBreakerNameKey]; name != "mytestname" || !ok {
			t.Errorf("bad breaker name")
			return false
		}
		if _, ok := p.Annotations[labeling.AnnotationBreakAtKey]; !ok {
			t.Errorf("missing breakAt annotation")
			return false
		}
		if retry, ok := p.Annotations[labeling.AnnotationRetryCountKey]; !ok || retry != "1" {
			t.Errorf("bad breakCount annotation")
			return false
		}
		return true
	}

	type fields struct {
		kubeClient    clientset.Interface
		breakerConfig v1.BreakerConfig
	}
	type args struct {
		inputPod *kapiv1.Pod
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		checkFunc func(*testing.T, *kapiv1.Pod) bool
		wantErr   bool
	}{
		{
			name: "update no Label",
			fields: fields{
				kubeClient: kfakeclient.NewSimpleClientset(test.PodGen("A", nil, true, true, "")),
				breakerConfig: v1.BreakerConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mytestname",
					},
				},
			},
			args: args{
				inputPod: test.PodGen("A", nil, true, true, ""),
			},
			checkFunc: checkFunc1,
			wantErr:   false,
		},
		{
			name: "update",
			fields: fields{
				kubeClient: kfakeclient.NewSimpleClientset(test.PodGen("A", nil, true, true, labeling.LabelTrafficYes)),
				breakerConfig: v1.BreakerConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mytestname",
					},
				},
			},
			args: args{
				inputPod: test.PodGen("A", nil, true, true, "labeling.LabelTrafficYes"),
			},
			checkFunc: checkFunc1,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Control{
				kubeClient:    tt.fields.kubeClient,
				breakerConfig: tt.fields.breakerConfig,
			}
			got, err := c.UpdateBreakerAnnotationAndLabel(tt.args.inputPod)

			if (err != nil) != tt.wantErr {
				t.Errorf("Control.UpdateBreakerAnnotationAndLabel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.checkFunc(t, got) {
				t.Errorf("Control.UpdateBreakerAnnotationAndLabel()")
			}
		})
	}
}

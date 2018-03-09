package pod

import (
	"testing"

	kapiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	kfakeclient "k8s.io/client-go/kubernetes/fake"

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
		kubeClient clientset.Interface
	}
	type args struct {
		breakerConfigName string
		inputPod          *kapiv1.Pod
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
				kubeClient: kfakeclient.NewSimpleClientset(test.PodGen("A", "test-ns", nil, true, true, "")),
			},
			args: args{
				breakerConfigName: "mytestname",
				inputPod:          test.PodGen("A", "test-ns", nil, true, true, ""),
			},
			checkFunc: checkFunc1,
			wantErr:   false,
		},
		{
			name: "update",
			fields: fields{
				kubeClient: kfakeclient.NewSimpleClientset(test.PodGen("A", "test-ns", nil, true, true, labeling.LabelTrafficYes)),
			},
			args: args{
				breakerConfigName: "mytestname",
				inputPod:          test.PodGen("A", "test-ns", nil, true, true, labeling.LabelTrafficYes),
			},
			checkFunc: checkFunc1,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Control{
				kubeClient: tt.fields.kubeClient,
			}
			got, err := c.UpdateBreakerAnnotationAndLabel(tt.args.breakerConfigName, tt.args.inputPod)

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

func TestControl_UpdateActivationLabelsAndAnnotations(t *testing.T) {

	checkFunc1 := func(t *testing.T, p *kapiv1.Pod) bool {
		if traffic, ok := p.Labels[labeling.LabelTrafficKey]; traffic != string(labeling.LabelTrafficYes) || !ok {
			t.Errorf("bad traffic label")
			return false
		}
		return true
	}

	type fields struct {
		kubeClient clientset.Interface
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
				kubeClient: kfakeclient.NewSimpleClientset(test.PodGen("A", "test-ns", nil, true, true, "")),
			},
			args: args{
				inputPod: test.PodGen("A", "test-ns", nil, true, true, ""),
			},
			checkFunc: checkFunc1,
			wantErr:   false,
		},
		{
			name: "update",
			fields: fields{
				kubeClient: kfakeclient.NewSimpleClientset(test.PodGen("A", "test-ns", nil, true, true, labeling.LabelTrafficNo)),
			},
			args: args{
				inputPod: test.PodGen("A", "test-ns", nil, true, true, labeling.LabelTrafficNo),
			},
			checkFunc: checkFunc1,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Control{
				kubeClient: tt.fields.kubeClient,
			}
			got, err := c.UpdateActivationLabelsAndAnnotations(tt.args.inputPod)

			if (err != nil) != tt.wantErr {
				t.Errorf("Control.UpdateActivationLabelsAndAnnotations() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.checkFunc(t, got) {
				t.Errorf("Control.UpdateActivationLabelsAndAnnotations()")
			}
		})
	}
}

func TestControl_UpdatePauseLabelsAndAnnotations(t *testing.T) {

	checkFunc1 := func(t *testing.T, p *kapiv1.Pod) bool {
		if traffic, ok := p.Labels[labeling.LabelTrafficKey]; traffic != string(labeling.LabelTrafficPause) || !ok {
			t.Errorf("bad traffic label")
			return false
		}
		return true
	}

	type fields struct {
		kubeClient clientset.Interface
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
				kubeClient: kfakeclient.NewSimpleClientset(test.PodGen("A", "test-ns", nil, true, true, "")),
			},
			args: args{
				inputPod: test.PodGen("A", "test-ns", nil, true, true, ""),
			},
			checkFunc: checkFunc1,
			wantErr:   false,
		},
		{
			name: "update",
			fields: fields{
				kubeClient: kfakeclient.NewSimpleClientset(test.PodGen("A", "test-ns", nil, true, true, labeling.LabelTrafficNo)),
			},
			args: args{
				inputPod: test.PodGen("A", "test-ns", nil, true, true, labeling.LabelTrafficNo),
			},
			checkFunc: checkFunc1,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Control{
				kubeClient: tt.fields.kubeClient,
			}
			got, err := c.UpdatePauseLabelsAndAnnotations(tt.args.inputPod)

			if (err != nil) != tt.wantErr {
				t.Errorf("Control.UpdatePauseLabelsAndAnnotations() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.checkFunc(t, got) {
				t.Errorf("Control.UpdatePauseLabelsAndAnnotations()")
			}
		})
	}
}

func TestControl_KillPod(t *testing.T) {

	checkFunc1 := func(t *testing.T, p *kapiv1.Pod, kc clientset.Interface) bool {
		if pod, err := kc.Core().Pods(p.Namespace).Get(p.Name, metav1.GetOptions{}); pod != nil || err == nil {
			t.Errorf("Should not find any pod")
			return false
		}
		return true
	}

	type fields struct {
		kubeClient clientset.Interface
	}
	type args struct {
		inputPod *kapiv1.Pod
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		checkFunc func(*testing.T, *kapiv1.Pod, clientset.Interface) bool
		wantErr   bool
	}{
		{
			name: "update no Label",
			fields: fields{
				kubeClient: kfakeclient.NewSimpleClientset(test.PodGen("A", "test-ns", nil, true, true, "")),
			},
			args: args{
				inputPod: test.PodGen("A", "test-ns", nil, true, true, ""),
			},
			checkFunc: checkFunc1,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Control{
				kubeClient: tt.fields.kubeClient,
			}
			if err := c.KillPod(tt.args.inputPod); (err != nil) != tt.wantErr {
				t.Errorf("Control.KillPod() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.checkFunc(t, tt.args.inputPod, tt.fields.kubeClient) {
				t.Errorf("Control.KillPod()")
			}
		})
	}
}

func TestControl_RemoveBreakerAnnotationAndLabel(t *testing.T) {
	checkFunc1 := func(t *testing.T, p *kapiv1.Pod) bool {
		if _, ok := p.Labels[labeling.LabelTrafficKey]; ok {
			t.Errorf("this label should not be present anymore! key:%s", labeling.LabelTrafficKey)
			return false
		}
		if _, ok := p.Labels[labeling.LabelBreakerNameKey]; ok {
			t.Errorf("this label should not be present anymore! key:%s", labeling.LabelBreakerNameKey)
			return false
		}
		if _, ok := p.Annotations[labeling.AnnotationBreakAtKey]; ok {
			t.Errorf("this label should not be present anymore! key:%s", labeling.AnnotationBreakAtKey)
			return false
		}
		if _, ok := p.Annotations[labeling.AnnotationRetryCountKey]; ok {
			t.Errorf("this label should not be present anymore! key:%s", labeling.AnnotationRetryCountKey)
			return false
		}
		return true
	}

	type fields struct {
		kubeClient clientset.Interface
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
			name: "not initialy present",
			fields: fields{
				kubeClient: kfakeclient.NewSimpleClientset(test.PodGen("A", "test-ns", nil, true, true, "")),
			},
			args: args{
				inputPod: test.PodGen("A", "test-ns", nil, true, true, ""),
			},
			checkFunc: checkFunc1,
			wantErr:   false,
		},
		{
			name: "remove labels and annotation",
			fields: fields{
				kubeClient: kfakeclient.NewSimpleClientset(test.PodGen("A", "test-ns", map[string]string{labeling.LabelTrafficKey: "yes", labeling.LabelBreakerNameKey: "foo"}, true, true, "")),
			},
			args: args{
				inputPod: test.PodGen("A", "test-ns", nil, true, true, ""),
			},
			checkFunc: checkFunc1,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Control{
				kubeClient: tt.fields.kubeClient,
			}
			got, err := c.RemoveBreakerAnnotationAndLabel(tt.args.inputPod)
			if (err != nil) != tt.wantErr {
				t.Errorf("Control.RemoveBreakerAnnotationAndLabel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.checkFunc(t, got) {
				t.Errorf("Control.RemoveBreakerAnnotationAndLabel() checkFunc return false, output pod:%v", got)
			}
		})
	}
}

package controller

import (
	"reflect"
	"testing"

	"github.com/amadeusitgroup/kubervisor/pkg/labeling"
	"github.com/amadeusitgroup/kubervisor/pkg/pod"
	test "github.com/amadeusitgroup/kubervisor/test"
	"go.uber.org/zap"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	kfakeclient "k8s.io/client-go/kubernetes/fake"
)

func TestController_searchNewPods(t *testing.T) {
	devlogger, _ := zap.NewDevelopment()
	newPod := test.PodGen("newPod", "test-ns", map[string]string{"app": "test-app"}, true, true, "")
	pod1 := test.PodGen("pod1", "test-ns", map[string]string{"app": "test-app", labeling.LabelTrafficKey: "yes", labeling.LabelBreakerNameKey: "foo"}, true, true, "")
	pod2 := test.PodGen("pod2", "test-ns", map[string]string{"app": "test-app", labeling.LabelTrafficKey: "yes", labeling.LabelBreakerNameKey: "foo"}, true, true, "")
	svc1 := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      "svc1",
		},
		Spec: apiv1.ServiceSpec{Selector: map[string]string{"app": "test-app"}},
	}
	type fields struct {
		kubeClient clientset.Interface
	}
	type args struct {
		svc *apiv1.Service
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*apiv1.Pod
		wantErr bool
	}{
		{
			name: "no new pods",
			fields: fields{
				kubeClient: kfakeclient.NewSimpleClientset(pod1, pod2),
			},
			args: args{
				svc: svc1,
			},
			want:    []*apiv1.Pod{},
			wantErr: false,
		},
		{
			name: "new pods",
			fields: fields{
				kubeClient: kfakeclient.NewSimpleClientset(newPod, pod1, pod2),
			},
			args: args{
				svc: svc1,
			},
			want:    []*apiv1.Pod{newPod},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := &Controller{
				Logger:     devlogger,
				kubeClient: tt.fields.kubeClient,
			}
			got, err := ctrl.searchNewPods(tt.args.svc)
			if (err != nil) != tt.wantErr {
				t.Errorf("Controller.searchNewPods() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Controller.searchNewPods() = %v \nwant %v", got, tt.want)
			}
		})
	}
}

func TestController_initializePods(t *testing.T) {
	devlogger, _ := zap.NewDevelopment()
	newPod := test.PodGen("newPod", "test-ns", map[string]string{"app": "test-app"}, true, true, "")
	pod1 := test.PodGen("pod1", "test-ns", map[string]string{"app": "test-app", labeling.LabelTrafficKey: "yes", labeling.LabelBreakerNameKey: "foo"}, true, true, "")
	pod2 := test.PodGen("pod2", "test-ns", map[string]string{"app": "test-app", labeling.LabelTrafficKey: "yes", labeling.LabelBreakerNameKey: "foo"}, true, true, "")
	svc1 := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      "svc1",
		},
		Spec: apiv1.ServiceSpec{Selector: map[string]string{"app": "test-app"}},
	}
	type fields struct {
		kubeClient clientset.Interface
		podControl pod.ControlInterface
	}
	type args struct {
		bciName string
		svc     *apiv1.Service
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "no new pods",
			fields: fields{
				kubeClient: kfakeclient.NewSimpleClientset(pod1, pod2),
				podControl: &test.TestPodControl{},
			},
			args: args{
				svc:     svc1,
				bciName: "foo",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "new pods",
			fields: fields{
				kubeClient: kfakeclient.NewSimpleClientset(newPod, pod1, pod2),
				podControl: &test.TestPodControl{},
			},
			args: args{
				svc: svc1,
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := &Controller{
				Logger:     devlogger,
				kubeClient: tt.fields.kubeClient,
				podControl: tt.fields.podControl,
			}
			got, err := ctrl.initializePods(tt.args.bciName, tt.args.svc)
			if (err != nil) != tt.wantErr {
				t.Errorf("Controller.initializePods() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Controller.initializePods() = %v, want %v", got, tt.want)
			}
		})
	}
}

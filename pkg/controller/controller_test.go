package controller

import (
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/labels"

	kubervisorapiv1 "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1"
	bclient "github.com/amadeusitgroup/kubervisor/pkg/client/clientset/versioned"
	"github.com/amadeusitgroup/kubervisor/pkg/client/clientset/versioned/fake"
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

type testInitializer struct {
	t               *testing.T
	nbWorker        uint32
	RegisterAPIFunc func() error
	InitClientsFunc func() (clientset.Interface, bclient.Interface, clientset.Interface, error)
}

var _ Initializer = &testInitializer{}

func (i *testInitializer) InitClients() (clientset.Interface, bclient.Interface, clientset.Interface, error) {
	if i.InitClientsFunc != nil {
		return i.InitClientsFunc()
	}
	kubeclient := kfakeclient.NewSimpleClientset()
	breakerClient := fake.NewSimpleClientset()
	return kubeclient, breakerClient, nil, nil
}
func (i *testInitializer) RegisterAPI() error {
	if i.RegisterAPIFunc != nil {
		return i.RegisterAPIFunc()
	}
	return nil
}
func (i *testInitializer) Logger() *zap.Logger {
	l, _ := zap.NewDevelopment()
	return l
}
func (i *testInitializer) NbWorker() uint32 {
	return i.nbWorker
}
func (i *testInitializer) HTTPServer() *http.Server {
	port, err := i.getFreePort()
	if err != nil {
		i.t.Fatalf("Can't get a free port: %v", err)
	}
	return &http.Server{Addr: ":" + port}
}

// GetFreePort asks the kernel for a free open port that is ready to use.
func (i *testInitializer) getFreePort() (string, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return "", err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return "", err
	}
	defer l.Close()
	return strconv.Itoa(l.Addr().(*net.TCPAddr).Port), nil
}

func TestNew(t *testing.T) {
	type args struct {
		initializer Initializer
	}
	tests := []struct {
		name    string
		args    args
		wantNil bool
	}{
		{
			name: "error registration",
			args: args{
				initializer: &testInitializer{
					t:        t,
					nbWorker: 1,
					RegisterAPIFunc: func() error {
						return fmt.Errorf("Bad Registration")
					},
				},
			},
			wantNil: true,
		},
		{
			name: "error clients",
			args: args{
				initializer: &testInitializer{
					t:        t,
					nbWorker: 1,
					InitClientsFunc: func() (clientset.Interface, bclient.Interface, clientset.Interface, error) {
						return nil, nil, nil, fmt.Errorf("Bad clients")
					},
				},
			},
			wantNil: true,
		},
		{
			name: "ok",
			args: args{
				initializer: &testInitializer{
					t:        t,
					nbWorker: 1,
				},
			},
			wantNil: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.args.initializer)
			if tt.wantNil && got != nil {
				t.Errorf("Want nil and got a controller")
			}
			if !tt.wantNil && got == nil {
				t.Errorf("Want a controller and got nil")
			}
		})
	}
}

func newTestController(t *testing.T) *Controller {
	initializer := &testInitializer{
		t:        t,
		nbWorker: 1,
	}
	ctrl := New(initializer)
	ctrl.updateHandlerFunc = func(bc *kubervisorapiv1.KubervisorService) (*kubervisorapiv1.KubervisorService, error) {
		ks, err := ctrl.breakerClient.Kubervisor().KubervisorServices(bc.Namespace).Update(bc)
		ctrl.breakerInformer.Informer().GetStore().Add(bc)
		ctrl.onAddKubervisorService(bc)
		return ks, err
	}
	return ctrl
}
func TestController_Run(t *testing.T) {
	ctrl := newTestController(t)
	stop := make(chan struct{})
	go func() {
		defer close(stop)
		pod1 := test.PodGen("pod1", "test-ns", map[string]string{"app": "test-app", labeling.LabelTrafficKey: "yes", labeling.LabelBreakerNameKey: "foo"}, true, true, "")
		svc1 := &apiv1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test-ns",
				Name:      "svc1",
			},
			Spec: apiv1.ServiceSpec{Selector: map[string]string{"app": "test-app"}},
		}
		svc2 := &apiv1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test-ns",
				Name:      "svc2",
			},
			Spec: apiv1.ServiceSpec{Selector: map[string]string{"app": "test-app"}},
		}
		if _, err := ctrl.kubeClient.CoreV1().Services("test-ns").Create(svc1); err != nil {
			t.Fatalf("Can't create service, err: %v", err)
			return
		}
		if _, err := ctrl.kubeClient.CoreV1().Services("test-ns").Create(svc2); err != nil {
			t.Fatalf("Can't create service, err: %v", err)
			return
		}
		if _, err := ctrl.kubeClient.CoreV1().Pods("test-ns").Create(pod1); err != nil {
			t.Fatalf("Can't create pods, err: %v", err)
			return
		}

		activatorStrategyConfig := kubervisorapiv1.DefaultActivatorStrategy(&kubervisorapiv1.ActivatorStrategy{})
		breakerStrategyConfig := &kubervisorapiv1.BreakerStrategy{ // Do not default to test the defaulting path.
			DiscreteValueOutOfList: &kubervisorapiv1.DiscreteValueOutOfList{
				PromQL:            "query",
				PrometheusService: "Service",
				GoodValues:        []string{"ok"},
				Key:               "code",
				PodNameKey:        "podname",
			},
		}
		bc := &kubervisorapiv1.KubervisorService{
			ObjectMeta: metav1.ObjectMeta{Name: "test-bc", Namespace: "test-ns"},
			Spec: kubervisorapiv1.KubervisorServiceSpec{
				DefaultActivator: *activatorStrategyConfig,
				Breakers:         []kubervisorapiv1.BreakerStrategy{*breakerStrategyConfig},
				Service:          "svc1",
			},
		}
		bcNoService := &kubervisorapiv1.KubervisorService{
			ObjectMeta: metav1.ObjectMeta{Name: "test-bc-noService", Namespace: "test-ns"},
			Spec: kubervisorapiv1.KubervisorServiceSpec{
				DefaultActivator: *activatorStrategyConfig,
				Breakers:         []kubervisorapiv1.BreakerStrategy{*breakerStrategyConfig},
				Service:          "noService",
			},
		}

		ksvc, err := ctrl.breakerClient.KubervisorV1().KubervisorServices("test-ns").Create(bc)
		if err != nil {
			t.Fatalf("Can't create kubervisor service, err: %v", err)
			return
		}
		ksvcnoService, err := ctrl.breakerClient.KubervisorV1().KubervisorServices("test-ns").Create(bcNoService)
		if err != nil {
			t.Fatalf("Can't create kubervisor service (with no service associated), err: %v", err)
			return
		}

		ctrl.breakerInformer.Informer().GetStore().Add(ksvc)
		ctrl.onAddKubervisorService(ksvc)
		ctrl.breakerInformer.Informer().GetStore().Add(ksvcnoService)
		ctrl.onAddKubervisorService(ksvcnoService)
		time.Sleep(1 * time.Second)

		ret, err := ctrl.breakerLister.List(labels.Everything())
		if err != nil {
			t.Fatalf("Can't list kubervisor service, err: %v", err)
		}

		if len(ret) != 2 {
			t.Fatalf("bad count kubervisorService 2!=%d", len(ret))
		}

		if len(ctrl.items.List()) != 1 {
			t.Fatalf("bad count for items 1!=%d", len(ctrl.items.List()))
		}

		//update spec
		{
			ksvcToUpdate, err := ctrl.breakerClient.KubervisorV1().KubervisorServices("test-ns").Get("test-bc", metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Can't retrieve kubervisor service, err: %v", err)
				return
			}
			oldkvs := ksvcToUpdate.DeepCopy()
			ksvcToUpdate.Spec.Breakers[0].MinPodsAvailableCount = kubervisorapiv1.NewUInt(100)
			ksvcToUpdate.Spec.Service = "svc2"
			ksvcUpdated, err := ctrl.breakerClient.KubervisorV1().KubervisorServices("test-ns").Update(ksvcToUpdate)
			if err != nil {
				t.Fatalf("Can't update kubervisor service, err: %v", err)
				return
			}
			ctrl.breakerInformer.Informer().GetStore().Update(ksvcUpdated)
			ctrl.onUpdateKubervisorService(oldkvs, ksvcUpdated)
			time.Sleep(1 * time.Second)
		}
		if len(ctrl.items.List()) != 1 {
			t.Fatalf("bad count for items (svc2) 1!=%d", len(ctrl.items.List()))
		}

		//change service
		{
			ksvcToUpdate, err := ctrl.breakerClient.KubervisorV1().KubervisorServices("test-ns").Get("test-bc", metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Can't retrieve kubervisor service, err: %v", err)
				return
			}
			oldkvs := ksvcToUpdate.DeepCopy()
			ksvcToUpdate.Spec.Service = "unknownService"
			ksvcUpdated, err := ctrl.breakerClient.KubervisorV1().KubervisorServices("test-ns").Update(ksvcToUpdate)
			if err != nil {
				t.Fatalf("Can't update kubervisor service, err: %v", err)
				return
			}
			ctrl.breakerInformer.Informer().GetStore().Update(ksvcUpdated)
			ctrl.onUpdateKubervisorService(oldkvs, ksvcUpdated)
			time.Sleep(1 * time.Second)
		}
		//check that items has been removed
		if len(ctrl.items.List()) != 0 {
			t.Fatalf("bad count for items 0!=%d", len(ctrl.items.List()))
		}

	}()
	if err := ctrl.Run(stop); err != nil {
		t.Fatalf("Blank run fail with error: %s", err)
	}
}

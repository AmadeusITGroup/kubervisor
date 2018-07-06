package controller

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"testing"
	"time"

	api "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1alpha1"
	bclient "github.com/amadeusitgroup/kubervisor/pkg/client/clientset/versioned"
	"github.com/amadeusitgroup/kubervisor/pkg/client/clientset/versioned/fake"
	"github.com/amadeusitgroup/kubervisor/pkg/controller/item"
	"github.com/amadeusitgroup/kubervisor/pkg/labeling"
	"github.com/amadeusitgroup/kubervisor/pkg/pod"
	test "github.com/amadeusitgroup/kubervisor/test"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

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
	addr, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
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
	ctrl.updateHandlerFunc = func(bc *api.KubervisorService) (*api.KubervisorService, error) {
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

		activatorStrategyConfig := api.DefaultActivatorStrategy(&api.ActivatorStrategy{})
		breakerStrategyConfig := &api.BreakerStrategy{ // Do not default to test the defaulting path.
			Name: "strategy1",
			DiscreteValueOutOfList: &api.DiscreteValueOutOfList{
				PromQL:            "query",
				PrometheusService: "Service",
				GoodValues:        []string{"ok"},
				Key:               "code",
				PodNameKey:        "podname",
			},
		}
		bc := &api.KubervisorService{
			ObjectMeta: metav1.ObjectMeta{Name: "test-bc", Namespace: "test-ns"},
			Spec: api.KubervisorServiceSpec{
				DefaultActivator: *activatorStrategyConfig,
				Breakers:         []api.BreakerStrategy{*breakerStrategyConfig},
				Service:          "svc1",
			},
		}
		bcNoService := &api.KubervisorService{
			ObjectMeta: metav1.ObjectMeta{Name: "test-bc-noService", Namespace: "test-ns"},
			Spec: api.KubervisorServiceSpec{
				DefaultActivator: *activatorStrategyConfig,
				Breakers:         []api.BreakerStrategy{*breakerStrategyConfig},
				Service:          "noService",
			},
		}

		ksvc, err := ctrl.breakerClient.KubervisorV1alpha1().KubervisorServices("test-ns").Create(bc)
		if err != nil {
			t.Fatalf("Can't create kubervisor service, err: %v", err)
			return
		}
		ksvcnoService, err := ctrl.breakerClient.KubervisorV1alpha1().KubervisorServices("test-ns").Create(bcNoService)
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
			ksvcToUpdate, err := ctrl.breakerClient.KubervisorV1alpha1().KubervisorServices("test-ns").Get("test-bc", metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Can't retrieve kubervisor service, err: %v", err)
				return
			}
			oldkvs := ksvcToUpdate.DeepCopy()
			ksvcToUpdate.Spec.Breakers[0].MinPodsAvailableCount = api.NewUInt(100)
			ksvcToUpdate.Spec.Service = "svc2"
			ksvcUpdated, err := ctrl.breakerClient.KubervisorV1alpha1().KubervisorServices("test-ns").Update(ksvcToUpdate)
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
			ksvcToUpdate, err := ctrl.breakerClient.KubervisorV1alpha1().KubervisorServices("test-ns").Get("test-bc", metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Can't retrieve kubervisor service, err: %v", err)
				return
			}
			oldkvs := ksvcToUpdate.DeepCopy()
			ksvcToUpdate.Spec.Service = "unknown-service"
			ksvcUpdated, err := ctrl.breakerClient.KubervisorV1alpha1().KubervisorServices("test-ns").Update(ksvcToUpdate)
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

func Test_deleteGauge(t *testing.T) {

	kubervisorGauges = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kubervisor_breaker_gauge",
			Help: "Display Pod under kubervisor management",
		},
		[]string{"name", "namespace", "type"}, // type={managed,breaked,paused,unknown}
	)

	type args struct {
		name      string
		namespace string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "value is not present",
			args: args{"foo", "bar"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleteGauge(tt.args.name, tt.args.namespace)
		})
	}
}

func TestController_clearItem(t *testing.T) {
	type fields struct {
		items     item.KubervisorServiceItemStore
		itemsList []item.Interface
	}
	type args struct {
		name      string
		namespace string
	}
	tests := []struct {
		name          string
		fields        fields
		args          args
		wantItemsSize int
	}{
		{
			name: "item exist",
			fields: fields{
				items:     item.NewBreackerConfigItemStore(),
				itemsList: []item.Interface{&fakeItem{name: "foo", namespace: "bar"}},
			},
			args: args{
				name:      "foo",
				namespace: "bar",
			},
			wantItemsSize: 0,
		},
		{
			name: "item doesnt exist",
			fields: fields{
				items:     item.NewBreackerConfigItemStore(),
				itemsList: []item.Interface{&fakeItem{name: "bob", namespace: "bar"}},
			},
			args: args{
				name:      "foo",
				namespace: "bar",
			},
			wantItemsSize: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := &Controller{
				items: tt.fields.items,
			}
			for _, item := range tt.fields.itemsList {
				ctrl.items.Add(item)
			}

			ctrl.clearItem(tt.args.name, tt.args.namespace)
			if size := len(ctrl.items.List()); size != tt.wantItemsSize {
				t.Errorf("[%s] wrong ctrl.items size: %d, wanted:%d", tt.name, size, tt.wantItemsSize)
			}
		})
	}
}

type fakeItem struct {
	name      string
	namespace string
}

func (f fakeItem) Name() string {
	return f.name
}
func (f fakeItem) Namespace() string {
	return f.namespace
}
func (f fakeItem) Start(ctx context.Context) {}
func (f fakeItem) Stop() error               { return nil }
func (f fakeItem) CompareWithSpec(spec *api.KubervisorServiceSpec, selector labels.Selector) bool {
	return true
}
func (f fakeItem) GetStatus() api.PodCountStatus {
	return api.PodCountStatus{}
}

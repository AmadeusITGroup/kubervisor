package item

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	api "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1alpha1"
	"github.com/amadeusitgroup/kubervisor/pkg/labeling"
	test "github.com/amadeusitgroup/kubervisor/test"
	"go.uber.org/zap"
	kapiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	kv1 "k8s.io/client-go/listers/core/v1"
)

func TestKubervisorServiceItemKeyFunc(t *testing.T) {
	tests := []struct {
		name    string
		obj     interface{}
		want    string
		wantErr bool
	}{
		{
			name: "cast_ok",
			obj: &KubervisorServiceItem{
				name:      "test1",
				namespace: "nstest1",
			},
			want:    "nstest1/test1",
			wantErr: false,
		},
		{
			name:    "cast_error",
			obj:     &struct{}{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := KubervisorServiceItemKeyFunc(tt.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("KubervisorServiceItemKeyFunc() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("KubervisorServiceItemKeyFunc() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKubervisorServiceItem_CompareWithSpec(t *testing.T) {
	activatorStrategyConfig := api.DefaultActivatorStrategy(&api.ActivatorStrategy{})
	breakerStrategyConfig := api.DefaultBreakerStrategy(&api.BreakerStrategy{
		Name:      "aname",
		Activator: activatorStrategyConfig,
		DiscreteValueOutOfList: &api.DiscreteValueOutOfList{
			PromQL:            "query",
			PrometheusService: "Service",
			GoodValues:        []string{"ok"},
			Key:               "code",
			PodNameKey:        "podname",
		},
	})
	bc := &api.KubervisorService{
		ObjectMeta: metav1.ObjectMeta{Name: "test-bc", Namespace: "test-ns"},
		Spec: api.KubervisorServiceSpec{
			DefaultActivator: *activatorStrategyConfig,
			Breakers:         []api.BreakerStrategy{*breakerStrategyConfig},
		},
	}
	cfg := &Config{
		PodLister: test.NewTestPodLister([]*kapiv1.Pod{}),
		Selector:  labels.Set(map[string]string{"app": "foo"}).AsSelector(),
	}

	type args struct {
		spec     *api.KubervisorServiceSpec
		selector labels.Selector
	}
	tests := []struct {
		name string
		init func() Interface
		args args
		want bool
	}{
		{
			name: "same",
			args: args{
				spec:     &bc.Spec,
				selector: labels.Set(map[string]string{"app": "foo"}).AsSelector(),
			},
			init: func() Interface {
				i, err := New(bc, cfg)
				if err != nil {
					t.Fatalf("Factory did not return an Interface: %v", err)
					return nil
				}
				return i
			},
			want: false,
		},
		{
			name: "no inner activator",
			args: args{
				spec:     &bc.Spec,
				selector: labels.Set(map[string]string{"app": "foo"}).AsSelector(),
			},
			init: func() Interface {
				bc2 := bc.DeepCopy()
				bc2.Spec.Breakers[0].Activator = nil
				i, err := New(bc2, cfg)
				if err != nil {
					t.Fatalf("Factory did not return an Interface: %v", err)
					return nil
				}
				return i
			},
			want: true,
		},
		{
			name: "different default activator",
			args: args{
				spec:     &bc.Spec,
				selector: labels.Set(map[string]string{"app": "foo"}).AsSelector(),
			},
			init: func() Interface {
				bc2 := bc.DeepCopy()
				bc2.Spec.DefaultActivator.Period = api.NewFloat64(12.04)
				i, err := New(bc2, cfg)
				if err != nil {
					t.Fatalf("Factory did not return an Interface: %v", err)
					return nil
				}
				return i
			},
			want: true,
		},
		{
			name: "different number of breakers",
			args: args{
				spec:     &bc.Spec,
				selector: labels.Set(map[string]string{"app": "foo"}).AsSelector(),
			},
			init: func() Interface {
				bc2 := bc.DeepCopy()
				secondStrategy := breakerStrategyConfig.DeepCopy()
				secondStrategy.Name = "other"
				bc2.Spec.Breakers = []api.BreakerStrategy{*breakerStrategyConfig, *secondStrategy}
				i, err := New(bc2, cfg)
				if err != nil {
					t.Fatalf("Factory did not return an Interface: %v", err)
					return nil
				}
				return i
			},
			want: true,
		},
		{
			name: "breaker strategy name change",
			args: args{
				spec:     &bc.Spec,
				selector: labels.Set(map[string]string{"app": "foo"}).AsSelector(),
			},
			init: func() Interface {
				bc2 := bc.DeepCopy()
				bc2.Spec.Breakers[0].Name = "newname"
				i, err := New(bc2, cfg)
				if err != nil {
					t.Fatalf("Factory did not return an Interface: %v", err)
					return nil
				}
				return i
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := tt.init()
			if b == nil {
				t.Fatalf("Factory did not return an Interface")
				return
			}
			if got := b.CompareWithSpec(tt.args.spec, tt.args.selector); got != tt.want {
				t.Errorf("KubervisorServiceItem.CompareWithSpec() = %v, want %v", got, tt.want)
			}
		})
	}
}

type fakeActivator struct {
	sync.Mutex
	sequence string
}

func (f *fakeActivator) addSequenceToken(token string) {
	f.Lock()
	defer f.Unlock()
	f.sequence += token
}
func (f *fakeActivator) CompareSequence(seq string) bool {
	f.Lock()
	defer f.Unlock()
	return seq == f.sequence
}
func (f *fakeActivator) Run(stop <-chan struct{}) {
	f.addSequenceToken("R")
	<-stop
	f.addSequenceToken("S")
}
func (f *fakeActivator) CompareConfig(specStrategy *api.ActivatorStrategy, specSelector labels.Selector) bool {
	return true
}
func (f *fakeActivator) GetStatus() api.PodCountStatus {
	return api.PodCountStatus{}
}

type fakeBreaker struct {
	sync.Mutex
	sequence string
}

func (f *fakeBreaker) addSequenceToken(token string) {
	f.Lock()
	defer f.Unlock()
	f.sequence += token
}

func (f *fakeBreaker) CompareSequence(seq string) bool {
	f.Lock()
	defer f.Unlock()
	return seq == f.sequence
}
func (f *fakeBreaker) Run(stop <-chan struct{}) {
	f.addSequenceToken("R")
	<-stop
	f.addSequenceToken("S")
}
func (f *fakeBreaker) CompareConfig(specConfig *api.BreakerStrategy, specSelector labels.Selector) bool {
	return true
}
func (f *fakeBreaker) Name() string { return "Name" }

func TestStartStop(t *testing.T) {
	// The failure of that test will consist in a timeout in case the sequence does not complete
	a := fakeActivator{}
	a1 := fakeActivator{}
	a2 := fakeActivator{}
	b1 := fakeBreaker{}
	b2 := fakeBreaker{}
	b3 := fakeBreaker{}

	item := KubervisorServiceItem{
		name:             "name",
		namespace:        "ns",
		defaultActivator: &a,
		breakers:         []breakerActivatorPair{{activator: &a1, breaker: &b1}, {activator: &a2, breaker: &b2}, {breaker: &b3}},
	}

	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()
	item.Start(ctx)
	for !func() bool {
		if a.CompareSequence("R") &&
			a1.CompareSequence("R") && b1.CompareSequence("R") &&
			a2.CompareSequence("R") && b2.CompareSequence("R") &&
			b3.CompareSequence("R") {
			return true
		}
		return false
	}() {
		time.Sleep(50 * time.Millisecond)
	}
	item.Stop()
	for !func() bool {
		if a.CompareSequence("RS") &&
			a1.CompareSequence("RS") && b1.CompareSequence("RS") &&
			a2.CompareSequence("RS") && b2.CompareSequence("RS") &&
			b3.CompareSequence("RS") {
			return true
		}
		return false
	}() {
		time.Sleep(50 * time.Millisecond)
	}
}

func TestStartCancel(t *testing.T) {
	// The failure of that test will consist in a timeout
	a := fakeActivator{}
	a1 := fakeActivator{}
	a2 := fakeActivator{}
	b1 := fakeBreaker{}
	b2 := fakeBreaker{}
	b3 := fakeBreaker{}

	item := KubervisorServiceItem{
		name:             "name",
		namespace:        "ns",
		defaultActivator: &a,
		breakers:         []breakerActivatorPair{{activator: &a1, breaker: &b1}, {activator: &a2, breaker: &b2}, {breaker: &b3}},
	}

	ctx, ctxCancel := context.WithCancel(context.Background())
	item.Start(ctx)
	for !func() bool {
		if a.CompareSequence("R") &&
			a1.CompareSequence("R") && b1.CompareSequence("R") &&
			a2.CompareSequence("R") && b2.CompareSequence("R") &&
			b3.CompareSequence("R") {
			return true
		}
		return false
	}() {
		time.Sleep(50 * time.Millisecond)
	}
	ctxCancel()
	for !func() bool {
		if a.CompareSequence("RS") &&
			a1.CompareSequence("RS") && b1.CompareSequence("RS") &&
			a2.CompareSequence("RS") && b2.CompareSequence("RS") &&
			b3.CompareSequence("RS") {
			return true
		}
		return false
	}() {
		time.Sleep(50 * time.Millisecond)
	}
}

func Test_GetStatus(t *testing.T) {
	ARunningReadyTraffic := test.PodGen("A", "test-ns", map[string]string{"app": "foo"}, nil, true, true, labeling.LabelTrafficYes)
	BRunningReadyTraffic := test.PodGen("B", "test-ns", map[string]string{"app": "foo"}, nil, true, true, labeling.LabelTrafficYes)
	CNotRunningReadyTraffic := test.PodGen("C", "test-ns", map[string]string{"app": "foo"}, nil, false, true, labeling.LabelTrafficYes)
	DRunningNotReadyPauseTraffic := test.PodGen("D", "test-ns", map[string]string{"app": "foo"}, nil, true, false, labeling.LabelTrafficPause)
	ERunningReadyNoTraffic := test.PodGen("E", "test-ns", map[string]string{"app": "foo"}, nil, true, true, labeling.LabelTrafficNo)
	PRunningReadyPauseTraffic := test.PodGen("P", "test-ns", map[string]string{"app": "foo"}, nil, true, true, labeling.LabelTrafficPause)
	BadAppRunningReadyTraffic := test.PodGen("BadApp", "test-ns", map[string]string{"app": "bar"}, nil, true, true, labeling.LabelTrafficYes)
	UnknowLabelTraffic := test.PodGen("A", "test-ns", map[string]string{"app": "foo"}, nil, true, true, "")
	type fields struct {
		selector  labels.Selector
		podLister kv1.PodNamespaceLister
		logger    *zap.Logger
	}
	tests := []struct {
		name    string
		fields  fields
		want    api.PodCountStatus
		wantErr error
	}{
		{
			name: "no pods",
			fields: fields{
				selector: labels.SelectorFromSet(map[string]string{"app": "foo"}),
				podLister: test.NewTestPodNamespaceLister(
					[]*kapiv1.Pod{},
					"test-ns",
				),
			},
			want: api.PodCountStatus{},
		},
		{
			name: "various pods",
			fields: fields{
				selector: labels.SelectorFromSet(map[string]string{"app": "foo"}),
				podLister: test.NewTestPodNamespaceLister(
					[]*kapiv1.Pod{
						ARunningReadyTraffic,
						BRunningReadyTraffic,
						CNotRunningReadyTraffic,
						DRunningNotReadyPauseTraffic,
						ERunningReadyNoTraffic,
						PRunningReadyPauseTraffic,
						BadAppRunningReadyTraffic,
						UnknowLabelTraffic,
					},
					"test-ns",
				),
			},
			want: api.PodCountStatus{
				NbPodsManaged: 4,
				NbPodsBreaked: 1,
				NbPodsPaused:  1,
				NbPodsUnknown: 1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &KubervisorServiceItem{
				selector:  tt.fields.selector,
				podLister: tt.fields.podLister,
			}
			got, gotErr := b.GetStatus()
			if tt.wantErr != gotErr {
				t.Errorf("GetStatus().error = %v, want %v", gotErr, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

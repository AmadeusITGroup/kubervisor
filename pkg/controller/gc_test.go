package controller

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	kapiv1 "k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1alpha1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor"
	"github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
	"github.com/amadeusitgroup/podkubervisor/pkg/client/clientset/versioned/fake"
	"github.com/amadeusitgroup/podkubervisor/pkg/client/informers/externalversions"
	blisters "github.com/amadeusitgroup/podkubervisor/pkg/client/listers/kubervisor/v1"
	"github.com/amadeusitgroup/podkubervisor/pkg/labeling"
	"github.com/amadeusitgroup/podkubervisor/pkg/pod"
	test "github.com/amadeusitgroup/podkubervisor/test"
)

func Test_newGarbageCollector(t *testing.T) {

	fakeKubervisorClient := fake.NewSimpleClientset()
	factory := externalversions.NewSharedInformerFactory(fakeKubervisorClient, time.Second)
	devlogger, _ := zap.NewDevelopment()

	type args struct {
		period            time.Duration
		podControl        pod.ControlInterface
		podLister         corev1listers.PodLister
		breakerLister     blisters.KubervisorServiceLister
		missCountBeforeGC int
		logger            *zap.Logger
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "nil",
			wantErr: true,
		},
		{
			name: "ok",
			args: args{
				period:            time.Second,
				podControl:        &test.TestPodControl{},
				podLister:         test.NewTestPodLister(nil),
				breakerLister:     factory.Breaker().V1().KubervisorServices().Lister(),
				missCountBeforeGC: 1,
				logger:            devlogger,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newGarbageCollector(tt.args.period, tt.args.podControl, tt.args.podLister, tt.args.breakerLister, tt.args.missCountBeforeGC, tt.args.logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("newGarbageCollector() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if got == nil {
				t.Errorf("newGarbageCollector() returned nil")
				return
			}
		})
	}
}

func Test_garbageCollector_updateCounters(t *testing.T) {

	ksvc := &v1.KubervisorService{}
	ksvc.Kind = "KubervisorService"
	ksvc.APIVersion = kubervisor.GroupName + "/" + v1alpha1.SchemeGroupVersion.Version
	ksvc.Name = "kkk"
	ksvc.Namespace = "test-ns"

	fakeKubervisorClient := fake.NewSimpleClientset(ksvc)
	factory := externalversions.NewSharedInformerFactory(fakeKubervisorClient, 10*time.Millisecond)
	devlogger, _ := zap.NewDevelopment()

	informer := factory.Breaker().V1().KubervisorServices()
	stCh := make(chan struct{})
	go informer.Informer().Run(stCh)
	cache.WaitForCacheSync(stCh, informer.Informer().HasSynced)
	defer close(stCh)

	A := test.PodGen("A", "test-ns-other", map[string]string{"app": "foo", labeling.LabelBreakerNameKey: "kkk"}, true, true, labeling.LabelTrafficYes)
	B := test.PodGen("B", "test-ns", map[string]string{"app": "foo", labeling.LabelBreakerNameKey: "kkk"}, true, false, labeling.LabelTrafficYes)
	C := test.PodGen("C", "test-ns", map[string]string{"app": "foo", labeling.LabelBreakerNameKey: "jjj"}, false, true, labeling.LabelTrafficYes)
	D := test.PodGen("D", "test-ns", nil, false, true, labeling.LabelTrafficYes)

	type fields struct {
		podLister     corev1listers.PodLister
		breakerLister blisters.KubervisorServiceLister
	}
	tests := []struct {
		name         string
		fields       fields
		expectedKeys []string
	}{
		{
			name: "PodsCount2",
			fields: fields{
				podLister:     test.NewTestPodLister([]*kapiv1.Pod{A, B, C, D}),
				breakerLister: informer.Lister(),
			},
			expectedKeys: []string{A.Namespace + "/" + A.Name, C.Namespace + "/" + C.Name},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gc := &garbageCollector{
				logger:            devlogger,
				podLister:         tt.fields.podLister,
				breakerLister:     tt.fields.breakerLister,
				period:            50 * time.Millisecond,
				missCountBeforeGC: 1,
				counters:          map[string]int{},
			}
			gc.updateCounters()
			if len(tt.expectedKeys) != len(gc.counters) {
				t.Fatalf("Bad counter map size %d!=%d", len(tt.expectedKeys), len(gc.counters))
			}
			for _, k := range tt.expectedKeys {
				if _, ok := gc.counters[k]; !ok {
					t.Fatalf("Missing key in counter map %s", k)
				}
			}
		})
	}
}

func Test_garbageCollector_cleanPods(t *testing.T) {
	devlogger, _ := zap.NewDevelopment()
	testprefix := t.Name()
	A := test.PodGen("A", "test-ns-other", map[string]string{"app": "foo", labeling.LabelBreakerNameKey: "kkk"}, true, true, labeling.LabelTrafficYes)
	B := test.PodGen("B", "test-ns", map[string]string{"app": "foo", labeling.LabelBreakerNameKey: "kkk"}, true, false, labeling.LabelTrafficYes)
	C := test.PodGen("C", "test-ns", map[string]string{"app": "foo", labeling.LabelBreakerNameKey: "jjj"}, false, true, labeling.LabelTrafficYes)
	D := test.PodGen("D", "test-ns", map[string]string{"app": "foo", labeling.LabelBreakerNameKey: "error"}, false, true, labeling.LabelTrafficYes)

	type fields struct {
		podLister         corev1listers.PodLister
		podControl        pod.ControlInterface
		missCountBeforeGC int
		counters          map[string]int
	}
	tests := []struct {
		name            string
		stepCount       int
		sequenceTimeout time.Duration
		fields          fields
	}{
		{
			name:            "check 2 calls on A and C",
			stepCount:       2,
			sequenceTimeout: 3 * time.Second,
			fields: fields{
				podLister: test.NewTestPodLister([]*kapiv1.Pod{A, B, C, D}),
				podControl: &test.TestPodControl{
					RemoveBreakerAnnotationAndLabelFunc: func(p *kapiv1.Pod) (*kapiv1.Pod, error) {
						if p == A {
							test.GetTestSequence(t, testprefix+"/check 2 calls on A and C").PassOnlyOnce(0)
						}
						if p == C {
							test.GetTestSequence(t, testprefix+"/check 2 calls on A and C").PassOnlyOnce(1)
						}
						if p == D {
							return nil, fmt.Errorf("Error for whatever reason")
						}
						return p, nil
					},
				},
				missCountBeforeGC: 1,
				counters:          map[string]int{"badkey": 1, "unknow/name": 1, A.Namespace + "/" + A.Name: 3, C.Namespace + "/" + C.Name: 1, D.Namespace + "/" + D.Name: 1},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sequence := test.NewTestSequence(t, testprefix+"/"+tt.name, tt.stepCount, tt.sequenceTimeout)
			gc := &garbageCollector{
				logger:            devlogger,
				podLister:         tt.fields.podLister,
				podControl:        tt.fields.podControl,
				missCountBeforeGC: tt.fields.missCountBeforeGC,
				counters:          tt.fields.counters,
			}
			go func() {
				time.Sleep(100 * time.Millisecond)
				gc.cleanPods()
			}()
			var wg sync.WaitGroup
			sequence.ValidateTestSequenceNoOrder(&wg)
			wg.Wait()
		})
	}
}

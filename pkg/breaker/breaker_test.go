package breaker

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	kapiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	kv1 "k8s.io/client-go/listers/core/v1"

	"github.com/amadeusitgroup/podkubervisor/pkg/anomalydetector"
	"github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
	"github.com/amadeusitgroup/podkubervisor/pkg/labeling"
	"github.com/amadeusitgroup/podkubervisor/pkg/pod"
	test "github.com/amadeusitgroup/podkubervisor/test"
)

func TestBreakerImpl_computeMinAvailablePods(t *testing.T) {
	type fields struct {
		breakerStrategyConfig v1.BreakerStrategy
	}
	type args struct {
		podUnderSelectorCount int
	}
	tests := []struct {
		name                  string
		fields                fields
		podUnderSelectorCount int
		want                  int
	}{
		{
			name: "zero",
			fields: fields{
				breakerStrategyConfig: v1.BreakerStrategy{},
			},
			podUnderSelectorCount: 10,
			want: 0,
		},
		{
			name: "count3",
			fields: fields{
				breakerStrategyConfig: v1.BreakerStrategy{
					MinPodsAvailableCount: v1.NewUInt(3),
				},
			},
			podUnderSelectorCount: 10,
			want: 3,
		},
		{
			name: "ratio5",
			fields: fields{
				breakerStrategyConfig: v1.BreakerStrategy{
					MinPodsAvailableRatio: v1.NewUInt(50),
				},
			},
			podUnderSelectorCount: 10,
			want: 5,
		},
		{
			name: "ratio5count3",
			fields: fields{
				breakerStrategyConfig: v1.BreakerStrategy{
					MinPodsAvailableCount: v1.NewUInt(3),
					MinPodsAvailableRatio: v1.NewUInt(50),
				},
			},
			podUnderSelectorCount: 10,
			want: 5,
		},
		{
			name: "ratio5count8",
			fields: fields{
				breakerStrategyConfig: v1.BreakerStrategy{
					MinPodsAvailableCount: v1.NewUInt(8),
					MinPodsAvailableRatio: v1.NewUInt(50),
				},
			},
			podUnderSelectorCount: 10,
			want: 8,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BreakerImpl{
				breakerStrategyConfig: tt.fields.breakerStrategyConfig,
			}
			if got := b.computeMinAvailablePods(tt.podUnderSelectorCount); got != tt.want {
				t.Errorf("BreakerImpl.computeMinAvailablePods() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBreakerImpl_Run(t *testing.T) {

	testprefix := t.Name()
	devlogger, _ := zap.NewDevelopment()

	ARunningReadyTraffic := test.PodGen("A", "test-ns", map[string]string{"app": "foo"}, true, true, labeling.LabelTrafficYes)
	BRunningReadyTraffic := test.PodGen("B", "test-ns", map[string]string{"app": "foo"}, true, true, labeling.LabelTrafficYes)
	CNotRunningReadyTraffic := test.PodGen("C", "test-ns", map[string]string{"app": "foo"}, false, true, labeling.LabelTrafficYes)
	DRunningNotReadyTraffic := test.PodGen("D", "test-ns", map[string]string{"app": "foo"}, true, false, labeling.LabelTrafficYes)
	ERunningNotReadyNoTraffic := test.PodGen("E", "test-ns", map[string]string{"app": "foo"}, true, false, labeling.LabelTrafficNo)
	NRunningReadyNoTraffic := test.PodGen("N", "test-ns", map[string]string{"app": "foo"}, true, true, labeling.LabelTrafficNo)
	BadAppRunningReadyTraffic := test.PodGen("BadApp", "test-ns", map[string]string{"app": "bar"}, true, true, labeling.LabelTrafficYes)

	type fields struct {
		breakerConfigName     string
		breakerStrategyConfig v1.BreakerStrategy
		selector              labels.Selector
		podLister             kv1.PodNamespaceLister
		podControl            pod.ControlInterface
		logger                *zap.Logger
		anomalyDetector       anomalydetector.AnomalyDetector
	}
	tests := []struct {
		name            string
		stepCount       int
		sequenceTimeout time.Duration
		fields          fields
		stop            chan struct{}
		wantErr         bool
	}{
		{
			name: "ok",
			fields: fields{
				breakerStrategyConfig: v1.BreakerStrategy{
					EvaluationPeriod:      v1.NewFloat64(0.05),
					MinPodsAvailableCount: v1.NewUInt(1),
				},
				selector: labels.SelectorFromSet(map[string]string{"app": "foo"}),
				podLister: test.NewTestPodNamespaceLister(
					[]*kapiv1.Pod{
						ARunningReadyTraffic,
						BRunningReadyTraffic,
						CNotRunningReadyTraffic,
						DRunningNotReadyTraffic,
						ERunningNotReadyNoTraffic,
						NRunningReadyNoTraffic,
						BadAppRunningReadyTraffic,
					},
					"test-ns",
				),
				logger:          devlogger,
				anomalyDetector: &testAnomalyDetector{pods: []*kapiv1.Pod{ARunningReadyTraffic}},
				podControl: &test.TestPodControl{
					UpdateBreakerAnnotationAndLabelFunc: func(name string, p *kapiv1.Pod) (*kapiv1.Pod, error) {
						if p == ARunningReadyTraffic {
							test.GetTestSequence(t, testprefix+"/ok").PassAtLeastOnce(0)
						} else {
							t.Fatalf("Bad Pod in test 'ok'")
						}
						return p, nil
					},
				},
			},
			stepCount:       1,
			sequenceTimeout: time.Second,
			stop:            make(chan struct{}),
			wantErr:         false,
		},
		{
			name: "cutall",
			fields: fields{
				breakerStrategyConfig: v1.BreakerStrategy{
					EvaluationPeriod:      v1.NewFloat64(0.05),
					MinPodsAvailableCount: v1.NewUInt(0),
					MinPodsAvailableRatio: v1.NewUInt(0),
				},
				selector: labels.SelectorFromSet(map[string]string{"app": "foo"}),
				podLister: test.NewTestPodNamespaceLister(
					[]*kapiv1.Pod{
						ARunningReadyTraffic,
						BRunningReadyTraffic,
						CNotRunningReadyTraffic,
						DRunningNotReadyTraffic,
						ERunningNotReadyNoTraffic,
						NRunningReadyNoTraffic,
						BadAppRunningReadyTraffic,
					},
					"test-ns",
				),
				logger:          devlogger,
				anomalyDetector: &testAnomalyDetector{pods: []*kapiv1.Pod{ARunningReadyTraffic, BRunningReadyTraffic}},
				podControl: &test.TestPodControl{
					UpdateBreakerAnnotationAndLabelFunc: func(name string, p *kapiv1.Pod) (*kapiv1.Pod, error) {
						switch p {
						case ARunningReadyTraffic:
							test.GetTestSequence(t, testprefix+"/cutall").PassAtLeastOnce(0)
						case BRunningReadyTraffic:
							test.GetTestSequence(t, testprefix+"/cutall").PassAtLeastOnce(1)
						default:
							t.Fatalf("Bad Pod in test 'cutall'")
						}
						return p, nil
					},
				},
			},
			stepCount:       2,
			sequenceTimeout: time.Second,
			stop:            make(chan struct{}),
			wantErr:         false,
		},
		{
			name: "bigCount",
			fields: fields{
				breakerStrategyConfig: v1.BreakerStrategy{
					EvaluationPeriod:      v1.NewFloat64(0.05),
					MinPodsAvailableCount: v1.NewUInt(10),
					MinPodsAvailableRatio: v1.NewUInt(0),
				},
				selector: labels.SelectorFromSet(map[string]string{"app": "foo"}),
				podLister: test.NewTestPodNamespaceLister(
					[]*kapiv1.Pod{
						ARunningReadyTraffic,
						BRunningReadyTraffic,
						CNotRunningReadyTraffic,
						DRunningNotReadyTraffic,
						ERunningNotReadyNoTraffic,
						NRunningReadyNoTraffic,
						BadAppRunningReadyTraffic,
					},
					"test-ns",
				),
				logger:          devlogger,
				anomalyDetector: &testAnomalyDetector{pods: []*kapiv1.Pod{ARunningReadyTraffic, BRunningReadyTraffic}},
				podControl: &test.TestPodControl{
					UpdateBreakerAnnotationAndLabelFunc: func(name string, p *kapiv1.Pod) (*kapiv1.Pod, error) {
						t.Fatalf("Test bigCount should not break any pod")
						return p, nil
					},
				},
			},
			stepCount:       0,
			sequenceTimeout: time.Second,
			stop:            make(chan struct{}),
			wantErr:         false,
		},
		{
			name: "0quota1cut2running",
			fields: fields{
				breakerStrategyConfig: v1.BreakerStrategy{
					EvaluationPeriod:      v1.NewFloat64(0.05),
					MinPodsAvailableCount: v1.NewUInt(0),
					MinPodsAvailableRatio: v1.NewUInt(0),
				},
				selector: labels.SelectorFromSet(map[string]string{"app": "foo"}),
				podLister: test.NewTestPodNamespaceLister(
					[]*kapiv1.Pod{
						ARunningReadyTraffic,
						BRunningReadyTraffic,
						CNotRunningReadyTraffic,
						DRunningNotReadyTraffic,
						ERunningNotReadyNoTraffic,
						NRunningReadyNoTraffic,
						BadAppRunningReadyTraffic,
					},
					"test-ns",
				),
				logger:          devlogger,
				anomalyDetector: &testAnomalyDetector{pods: []*kapiv1.Pod{ARunningReadyTraffic}},
				podControl: &test.TestPodControl{
					UpdateBreakerAnnotationAndLabelFunc: func(name string, p *kapiv1.Pod) (*kapiv1.Pod, error) {
						switch p {
						case ARunningReadyTraffic:
							test.GetTestSequence(t, testprefix+"/0quota1cut2running").PassAtLeastOnce(0)
						default:
							t.Fatalf("Test '0quota1cut2running' should break pod A")
						}
						return p, nil
					},
				},
			},
			stepCount:       1,
			sequenceTimeout: time.Second,
			stop:            make(chan struct{}),
			wantErr:         false,
		},
		{
			name: "only1",
			fields: fields{
				breakerStrategyConfig: v1.BreakerStrategy{
					EvaluationPeriod:      v1.NewFloat64(0.05),
					MinPodsAvailableCount: v1.NewUInt(1),
					MinPodsAvailableRatio: v1.NewUInt(0),
				},
				selector: labels.SelectorFromSet(map[string]string{"app": "foo"}),
				podLister: test.NewTestPodNamespaceLister(
					[]*kapiv1.Pod{
						ARunningReadyTraffic,
						BRunningReadyTraffic,
						CNotRunningReadyTraffic,
						DRunningNotReadyTraffic,
						ERunningNotReadyNoTraffic,
						NRunningReadyNoTraffic,
						BadAppRunningReadyTraffic,
					},
					"test-ns",
				),
				logger:          devlogger,
				anomalyDetector: &testAnomalyDetector{pods: []*kapiv1.Pod{ARunningReadyTraffic, BRunningReadyTraffic}},
				podControl: &test.TestPodControl{
					UpdateBreakerAnnotationAndLabelFunc: func(name string, p *kapiv1.Pod) (*kapiv1.Pod, error) {
						switch p {
						case ARunningReadyTraffic:
							test.GetTestSequence(t, testprefix+"/only1").PassAtLeastOnce(0)
						default:
							t.Fatalf("B pod of test only1 should not be break")
						}
						return p, nil
					},
				},
			},
			stepCount:       1,
			sequenceTimeout: time.Second,
			stop:            make(chan struct{}),
			wantErr:         false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer close(tt.stop)
			sequence := test.NewTestSequence(t, testprefix+"/"+tt.name, tt.stepCount, tt.sequenceTimeout)
			b := &BreakerImpl{
				breakerStrategyConfig: tt.fields.breakerStrategyConfig,
				selector:              tt.fields.selector,
				podLister:             tt.fields.podLister,
				podControl:            tt.fields.podControl,
				logger:                tt.fields.logger,
				anomalyDetector:       tt.fields.anomalyDetector,
			}
			go b.Run(tt.stop)
			var wg sync.WaitGroup
			sequence.ValidateTestSequenceNoOrder(&wg)
			wg.Wait()

			if tt.stepCount == 0 {
				//Sleep to be sure that the breaker has some time to do several loops
				time.Sleep(time.Duration(5000*(*tt.fields.breakerStrategyConfig.EvaluationPeriod)) * time.Millisecond)
			}
		})
	}
	time.Sleep(time.Second)
}

type testAnomalyDetector struct {
	pods     []*kapiv1.Pod
	errOnce  error
	nilOnce  bool
	zeroOnce bool
}

func (t *testAnomalyDetector) GetPodsOutOfBounds() ([]*kapiv1.Pod, error) {
	if t.errOnce == nil {
		t.errOnce = fmt.Errorf("Error Once")
		return nil, t.errOnce
	}
	if !t.nilOnce {
		t.nilOnce = true
		return nil, nil
	}
	if !t.zeroOnce {
		t.zeroOnce = true
		return []*kapiv1.Pod{}, nil
	}
	return t.pods, nil
}

func TestBreakerImpl_CompareConfig(t *testing.T) {
	type fields struct {
		KubervisorServiceName string
		breakerStrategyConfig v1.BreakerStrategy
		selector              labels.Selector
		podLister             kv1.PodNamespaceLister
		podControl            pod.ControlInterface
		logger                *zap.Logger
		anomalyDetector       anomalydetector.AnomalyDetector
	}
	type args struct {
		specConfig *v1.BreakerStrategy
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{

		{
			name: "similar",
			fields: fields{
				breakerStrategyConfig: *v1.DefaultBreakerStrategy(&v1.BreakerStrategy{}),
			},
			args: args{
				specConfig: v1.DefaultBreakerStrategy(&v1.BreakerStrategy{}),
			},
			want: true,
		},
		{
			name: "different",
			fields: fields{
				breakerStrategyConfig: *v1.DefaultBreakerStrategy(&v1.BreakerStrategy{}),
			},
			args: args{
				specConfig: v1.DefaultBreakerStrategy(&v1.BreakerStrategy{EvaluationPeriod: v1.NewFloat64(42)}),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BreakerImpl{
				KubervisorServiceName: tt.fields.KubervisorServiceName,
				breakerStrategyConfig: tt.fields.breakerStrategyConfig,
				selector:              tt.fields.selector,
				podLister:             tt.fields.podLister,
				podControl:            tt.fields.podControl,
				logger:                tt.fields.logger,
				anomalyDetector:       tt.fields.anomalyDetector,
			}
			if got := b.CompareConfig(tt.args.specConfig); got != tt.want {
				t.Errorf("BreakerImpl.CompareConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBreakerImpl_GetStatus(t *testing.T) {
	ARunningReadyTraffic := test.PodGen("A", "test-ns", map[string]string{"app": "foo"}, true, true, labeling.LabelTrafficYes)
	BRunningReadyTraffic := test.PodGen("B", "test-ns", map[string]string{"app": "foo"}, true, true, labeling.LabelTrafficYes)
	CNotRunningReadyTraffic := test.PodGen("C", "test-ns", map[string]string{"app": "foo"}, false, true, labeling.LabelTrafficYes)
	DRunningNotReadyPauseTraffic := test.PodGen("D", "test-ns", map[string]string{"app": "foo"}, true, false, labeling.LabelTrafficPause)
	ERunningReadyNoTraffic := test.PodGen("E", "test-ns", map[string]string{"app": "foo"}, true, true, labeling.LabelTrafficNo)
	PRunningReadyPauseTraffic := test.PodGen("P", "test-ns", map[string]string{"app": "foo"}, true, true, labeling.LabelTrafficPause)
	BadAppRunningReadyTraffic := test.PodGen("BadApp", "test-ns", map[string]string{"app": "bar"}, true, true, labeling.LabelTrafficYes)
	UnknowLabelTraffic := test.PodGen("A", "test-ns", map[string]string{"app": "foo"}, true, true, "")
	type fields struct {
		KubervisorServiceName string
		breakerStrategyConfig v1.BreakerStrategy
		selector              labels.Selector
		podLister             kv1.PodNamespaceLister
		podControl            pod.ControlInterface
		logger                *zap.Logger
		anomalyDetector       anomalydetector.AnomalyDetector
	}
	tests := []struct {
		name   string
		fields fields
		want   v1.BreakerStatus
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
			want: v1.BreakerStatus{},
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
			want: v1.BreakerStatus{
				NbPodsManaged: 4,
				NbPodsBreaked: 1,
				NbPodsPaused:  1,
				NbPodsUnknown: 1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BreakerImpl{
				KubervisorServiceName: tt.fields.KubervisorServiceName,
				breakerStrategyConfig: tt.fields.breakerStrategyConfig,
				selector:              tt.fields.selector,
				podLister:             tt.fields.podLister,
				podControl:            tt.fields.podControl,
				logger:                tt.fields.logger,
				anomalyDetector:       tt.fields.anomalyDetector,
			}
			if got := b.GetStatus(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BreakerImpl.GetStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

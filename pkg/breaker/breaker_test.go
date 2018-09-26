package breaker

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	kapiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	kv1 "k8s.io/client-go/listers/core/v1"

	"github.com/amadeusitgroup/kubervisor/pkg/anomalydetector"
	api "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1alpha1"
	"github.com/amadeusitgroup/kubervisor/pkg/labeling"
	"github.com/amadeusitgroup/kubervisor/pkg/pod"
	test "github.com/amadeusitgroup/kubervisor/test"
)

func TestBreakerImpl_computeMinAvailablePods(t *testing.T) {
	type fields struct {
		breakerStrategyConfig api.BreakerStrategy
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
				breakerStrategyConfig: api.BreakerStrategy{},
			},
			podUnderSelectorCount: 10,
			want: 0,
		},
		{
			name: "count3",
			fields: fields{
				breakerStrategyConfig: api.BreakerStrategy{
					MinPodsAvailableCount: api.NewUInt(3),
				},
			},
			podUnderSelectorCount: 10,
			want: 3,
		},
		{
			name: "ratio5",
			fields: fields{
				breakerStrategyConfig: api.BreakerStrategy{
					MinPodsAvailableRatio: api.NewUInt(50),
				},
			},
			podUnderSelectorCount: 10,
			want: 5,
		},
		{
			name: "ratio5count3",
			fields: fields{
				breakerStrategyConfig: api.BreakerStrategy{
					MinPodsAvailableCount: api.NewUInt(3),
					MinPodsAvailableRatio: api.NewUInt(50),
				},
			},
			podUnderSelectorCount: 10,
			want: 5,
		},
		{
			name: "ratio5count8",
			fields: fields{
				breakerStrategyConfig: api.BreakerStrategy{
					MinPodsAvailableCount: api.NewUInt(8),
					MinPodsAvailableRatio: api.NewUInt(50),
				},
			},
			podUnderSelectorCount: 10,
			want: 8,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &breakerImpl{
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

	ARunningReadyTraffic := test.PodGen("A", "test-ns", map[string]string{"app": "foo"}, nil, true, true, labeling.LabelTrafficYes)
	BRunningReadyTraffic := test.PodGen("B", "test-ns", map[string]string{"app": "foo"}, nil, true, true, labeling.LabelTrafficYes)
	CNotRunningReadyTraffic := test.PodGen("C", "test-ns", map[string]string{"app": "foo"}, nil, false, true, labeling.LabelTrafficYes)
	DRunningNotReadyTraffic := test.PodGen("D", "test-ns", map[string]string{"app": "foo"}, nil, true, false, labeling.LabelTrafficYes)
	ERunningNotReadyNoTraffic := test.PodGen("E", "test-ns", map[string]string{"app": "foo"}, nil, true, false, labeling.LabelTrafficNo)
	NRunningReadyNoTraffic := test.PodGen("N", "test-ns", map[string]string{"app": "foo"}, nil, true, true, labeling.LabelTrafficNo)
	BadAppRunningReadyTraffic := test.PodGen("BadApp", "test-ns", map[string]string{"app": "bar"}, nil, true, true, labeling.LabelTrafficYes)

	type fields struct {
		breakerConfigName     string
		breakerStrategyConfig api.BreakerStrategy
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
				breakerStrategyConfig: api.BreakerStrategy{
					EvaluationPeriod:      api.NewFloat64(0.05),
					MinPodsAvailableCount: api.NewUInt(1),
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
					UpdateBreakerAnnotationAndLabelFunc: func(name string, strategy string, p *kapiv1.Pod) (*kapiv1.Pod, error) {
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
				breakerStrategyConfig: api.BreakerStrategy{
					EvaluationPeriod:      api.NewFloat64(0.05),
					MinPodsAvailableCount: api.NewUInt(0),
					MinPodsAvailableRatio: api.NewUInt(0),
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
					UpdateBreakerAnnotationAndLabelFunc: func(name string, strategy string, p *kapiv1.Pod) (*kapiv1.Pod, error) {
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
				breakerStrategyConfig: api.BreakerStrategy{
					EvaluationPeriod:      api.NewFloat64(0.05),
					MinPodsAvailableCount: api.NewUInt(10),
					MinPodsAvailableRatio: api.NewUInt(0),
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
					UpdateBreakerAnnotationAndLabelFunc: func(name string, strategy string, p *kapiv1.Pod) (*kapiv1.Pod, error) {
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
				breakerStrategyConfig: api.BreakerStrategy{
					EvaluationPeriod:      api.NewFloat64(0.05),
					MinPodsAvailableCount: api.NewUInt(0),
					MinPodsAvailableRatio: api.NewUInt(0),
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
					UpdateBreakerAnnotationAndLabelFunc: func(name string, strategy string, p *kapiv1.Pod) (*kapiv1.Pod, error) {
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
				breakerStrategyConfig: api.BreakerStrategy{
					EvaluationPeriod:      api.NewFloat64(0.05),
					MinPodsAvailableCount: api.NewUInt(1),
					MinPodsAvailableRatio: api.NewUInt(0),
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
					UpdateBreakerAnnotationAndLabelFunc: func(name string, strategy string, p *kapiv1.Pod) (*kapiv1.Pod, error) {
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
			b := &breakerImpl{
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
		breakerName           string
		breakerStrategyConfig api.BreakerStrategy
		selector              labels.Selector
		podLister             kv1.PodNamespaceLister
		podControl            pod.ControlInterface
		logger                *zap.Logger
		anomalyDetector       anomalydetector.AnomalyDetector
	}
	type args struct {
		specConfig   *api.BreakerStrategy
		specSelector labels.Selector
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
				breakerStrategyConfig: *api.DefaultBreakerStrategy(&api.BreakerStrategy{}),
				breakerName:           "b1",
				selector:              labels.Set{"app": "test1", labeling.LabelBreakerNameKey: "b1"}.AsSelectorPreValidated(),
			},
			args: args{
				specConfig:   api.DefaultBreakerStrategy(&api.BreakerStrategy{}),
				specSelector: labels.Set{"app": "test1"}.AsSelectorPreValidated(),
			},
			want: true,
		},
		{
			name: "different",
			fields: fields{
				breakerStrategyConfig: *api.DefaultBreakerStrategy(&api.BreakerStrategy{}),
				breakerName:           "b1",
				selector:              labels.Set{"app": "test1", labeling.LabelBreakerNameKey: "b1"}.AsSelectorPreValidated(),
			},
			args: args{
				specConfig:   api.DefaultBreakerStrategy(&api.BreakerStrategy{EvaluationPeriod: api.NewFloat64(42)}),
				specSelector: labels.Set{"app": "test1"}.AsSelectorPreValidated(),
			},
			want: false,
		},
		{
			name: "different labels",
			fields: fields{
				breakerStrategyConfig: *api.DefaultBreakerStrategy(&api.BreakerStrategy{}),
				breakerName:           "b1",
				selector:              labels.Set{"app": "test1", labeling.LabelBreakerNameKey: "b1"}.AsSelectorPreValidated(),
			},
			args: args{
				specConfig:   api.DefaultBreakerStrategy(&api.BreakerStrategy{}),
				specSelector: labels.Set{"app": "test2"}.AsSelectorPreValidated(),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &breakerImpl{
				breakerStrategyConfig: tt.fields.breakerStrategyConfig,
				kubervisorName:        tt.fields.breakerName,
				selector:              tt.fields.selector,
				podLister:             tt.fields.podLister,
				podControl:            tt.fields.podControl,
				logger:                tt.fields.logger,
				anomalyDetector:       tt.fields.anomalyDetector,
			}
			if got := b.CompareConfig(tt.args.specConfig, tt.args.specSelector); got != tt.want {
				t.Errorf("BreakerImpl.CompareConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

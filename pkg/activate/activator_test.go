package activate

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	kapiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	kv1 "k8s.io/client-go/listers/core/v1"

	"github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
	"github.com/amadeusitgroup/podkubervisor/pkg/labeling"
	"github.com/amadeusitgroup/podkubervisor/pkg/pod"
	test "github.com/amadeusitgroup/podkubervisor/test"
)

type testStrategyApplier struct {
	applyFunc func(p *kapiv1.Pod) error
}

func (t *testStrategyApplier) applyActivatorStrategy(p *kapiv1.Pod) error {
	if t.applyFunc != nil {
		return t.applyFunc(p)
	}
	return nil
}

var _ strategyApplier = &testStrategyApplier{}

func TestActivatorImpl_Run(t *testing.T) {
	testprefix := t.Name()
	devlogger, _ := zap.NewDevelopment()

	type fields struct {
		activatorStrategyConfig v1.ActivatorStrategy
		selector                labels.Selector
		podLister               kv1.PodLister
		podControl              pod.ControlInterface
		breakerName             string
		logger                  *zap.Logger
		strategyApplier         strategyApplier
	}
	type args struct {
	}
	tests := []struct {
		name            string
		stepCount       int
		sequenceTimeout time.Duration
		fields          fields
		args            args
		stop            chan struct{}
	}{
		{
			name: "2pods",
			fields: fields{
				selector: labels.SelectorFromSet(map[string]string{"app": "foo"}),
				podLister: test.NewTestPodLister(
					[]*kapiv1.Pod{
						test.PodGen("A", map[string]string{"app": "foo"}, true, true, labeling.LabelTrafficNo),
						test.PodGen("AA", map[string]string{"app": "foo"}, true, true, labeling.LabelTrafficNo),
						test.PodGen("B", map[string]string{"app": "foo"}, true, true, labeling.LabelTrafficYes),
						test.PodGen("C", map[string]string{"app": "foo"}, true, true, labeling.LabelTrafficPause),
						test.PodGen("D", map[string]string{"app": "foo"}, true, true, ""),
						test.PodGen("E", map[string]string{"app": "other"}, true, true, labeling.LabelTrafficNo),
					}),
				breakerName: "2pods",
				logger:      devlogger,
				strategyApplier: &testStrategyApplier{
					applyFunc: func(p *kapiv1.Pod) error {
						switch p.Name {
						case "A":
							test.GetTestSequence(t, testprefix+"/2pods").PassAtLeastOnce(0)
						case "AA":
							test.GetTestSequence(t, testprefix+"/2pods").PassAtLeastOnce(1)
							return fmt.Errorf("error case")
						default:
							t.Fatalf("Unexpected pod %s", p.Name)
						}
						return nil
					},
				},
			},
			stop:            make(chan struct{}),
			stepCount:       2,
			sequenceTimeout: time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer close(tt.stop)
			sequence := test.NewTestSequence(t, testprefix+"/"+tt.name, tt.stepCount, tt.sequenceTimeout)
			b := &ActivatorImpl{
				activatorStrategyConfig: tt.fields.activatorStrategyConfig,
				selector:                tt.fields.selector,
				podLister:               tt.fields.podLister,
				podControl:              tt.fields.podControl,
				breakerName:             tt.fields.breakerName,
				logger:                  tt.fields.logger,
				evaluationPeriod:        20 * time.Millisecond,
				strategyApplier:         tt.fields.strategyApplier,
			}
			go b.Run(tt.stop)

			var wg sync.WaitGroup
			sequence.ValidateTestSequenceNoOrder(&wg)
			wg.Wait()

			if sequence.Len() == 0 {
				//Sleep to be sure that the breaker has some time to do several loops
				time.Sleep(time.Duration(5) * b.evaluationPeriod)
			}
		})
	}
}

func TestActivatorImpl_applyActivatorStrategy(t *testing.T) {
	testprefix := t.Name()
	type fields struct {
		activatorStrategyConfig v1.ActivatorStrategy
		selector                labels.Selector
		podLister               kv1.PodLister
		podControl              pod.ControlInterface
		breakerName             string
		logger                  *zap.Logger
	}

	tests := []struct {
		name     string
		fields   fields
		inputPod func() *kapiv1.Pod
		wantErr  bool
		sequence *test.TestStepSequence
	}{
		{
			name:     "missingBreatAtAnnotation",
			inputPod: func() *kapiv1.Pod { return test.PodGen("A", nil, true, true, "") },
			wantErr:  true,
		},
		{
			name: "missingRetryCountAnnotation",
			inputPod: func() *kapiv1.Pod {
				p := test.PodGen("A", nil, true, true, "")
				p.Annotations = map[string]string{labeling.AnnotationBreakAtKey: string("1978-12-04T22:11:00+00:00")}
				return p
			},
			wantErr: true,
		},
		{
			name: "periodic_not_trigerred",
			fields: fields{
				activatorStrategyConfig: v1.ActivatorStrategy{
					Mode: v1.ActivatorStrategyModePeriodic,
				},
				podControl: &test.TestPodControl{
					T:                   t,
					FailOnUndefinedFunc: true,
				},
			},
			inputPod: func() *kapiv1.Pod {
				p := test.PodGen("A", nil, true, true, "")
				p.Annotations = map[string]string{
					labeling.AnnotationBreakAtKey:    string("2078-12-04T22:11:00+00:00"),
					labeling.AnnotationRetryCountKey: "1",
				}
				return p
			},
			wantErr: false,
		},
		{
			name: "periodic_with_error",
			fields: fields{
				activatorStrategyConfig: v1.ActivatorStrategy{
					Mode: v1.ActivatorStrategyModePeriodic,
				},
				podControl: &test.TestPodControl{
					T:                   t,
					FailOnUndefinedFunc: true,
					UpdateActivationLabelsAndAnnotationsFunc: func(p *kapiv1.Pod) (*kapiv1.Pod, error) {
						return nil, fmt.Errorf("Fake Error")
					},
				},
			},
			inputPod: func() *kapiv1.Pod {
				p := test.PodGen("A", nil, true, true, "")
				p.Annotations = map[string]string{
					labeling.AnnotationBreakAtKey:    string("1978-12-04T22:11:00+00:00"),
					labeling.AnnotationRetryCountKey: "1",
				}
				return p
			},
			wantErr: true,
		},
		{
			name:     "periodic_fine",
			sequence: test.NewTestSequence(t, testprefix+"/periodic_fine", 1, time.Second),
			fields: fields{
				activatorStrategyConfig: v1.ActivatorStrategy{
					Mode: v1.ActivatorStrategyModePeriodic,
				},
				podControl: &test.TestPodControl{
					T:                   t,
					FailOnUndefinedFunc: true,
					UpdateActivationLabelsAndAnnotationsFunc: func(p *kapiv1.Pod) (*kapiv1.Pod, error) {
						test.GetTestSequence(t, testprefix+"/periodic_fine").PassAtLeastOnce(0)
						return p, nil
					},
				},
			},
			inputPod: func() *kapiv1.Pod {
				p := test.PodGen("A", nil, true, true, "")
				p.Annotations = map[string]string{
					labeling.AnnotationBreakAtKey:    string("1978-12-04T22:11:00+00:00"),
					labeling.AnnotationRetryCountKey: "1",
				}
				return p
			},
			wantErr: false,
		},
		{
			name: "retryAndKill_kill_error",
			fields: fields{
				activatorStrategyConfig: v1.ActivatorStrategy{
					Mode:          v1.ActivatorStrategyModeRetryAndKill,
					MaxRetryCount: v1.NewUInt(3),
				},
				podControl: &test.TestPodControl{
					T:                   t,
					FailOnUndefinedFunc: true,
					KillPodFunc: func(p *kapiv1.Pod) error {
						return fmt.Errorf("Fake Error")
					},
				},
			},
			inputPod: func() *kapiv1.Pod {
				p := test.PodGen("A", nil, true, true, "")
				p.Annotations = map[string]string{
					labeling.AnnotationBreakAtKey:    string("1978-12-04T22:11:00+00:00"),
					labeling.AnnotationRetryCountKey: "4",
				}
				return p
			},
			wantErr: true,
		},
		{
			name:     "retryAndKill_kill_fine",
			sequence: test.NewTestSequence(t, testprefix+"/retryAndKill_kill_fine", 1, time.Second),
			fields: fields{
				activatorStrategyConfig: v1.ActivatorStrategy{
					Mode:          v1.ActivatorStrategyModeRetryAndKill,
					MaxRetryCount: v1.NewUInt(3),
				},
				podControl: &test.TestPodControl{
					T:                   t,
					FailOnUndefinedFunc: true,
					KillPodFunc: func(p *kapiv1.Pod) error {
						test.GetTestSequence(t, testprefix+"/retryAndKill_kill_fine").PassAtLeastOnce(0)
						return nil
					},
				},
			},
			inputPod: func() *kapiv1.Pod {
				p := test.PodGen("A", nil, true, true, "")
				p.Annotations = map[string]string{
					labeling.AnnotationBreakAtKey:    string("1978-12-04T22:11:00+00:00"),
					labeling.AnnotationRetryCountKey: "4",
				}
				return p
			},
			wantErr: false,
		},
		{
			name: "retryAndKill_retry_error",
			fields: fields{
				activatorStrategyConfig: v1.ActivatorStrategy{
					Mode:          v1.ActivatorStrategyModeRetryAndKill,
					MaxRetryCount: v1.NewUInt(3),
				},
				podControl: &test.TestPodControl{
					T:                   t,
					FailOnUndefinedFunc: true,
					UpdateActivationLabelsAndAnnotationsFunc: func(p *kapiv1.Pod) (*kapiv1.Pod, error) {
						return nil, fmt.Errorf("fake Error")
					},
				},
			},
			inputPod: func() *kapiv1.Pod {
				p := test.PodGen("A", nil, true, true, "")
				p.Annotations = map[string]string{
					labeling.AnnotationBreakAtKey:    string("1978-12-04T22:11:00+00:00"),
					labeling.AnnotationRetryCountKey: "2",
				}
				return p
			},
			wantErr: true,
		},
		{
			name:     "retryAndKill_retry_fine",
			sequence: test.NewTestSequence(t, testprefix+"/retryAndKill_retry_fine", 1, time.Second),
			fields: fields{
				activatorStrategyConfig: v1.ActivatorStrategy{
					Mode:          v1.ActivatorStrategyModeRetryAndKill,
					MaxRetryCount: v1.NewUInt(3),
				},
				podControl: &test.TestPodControl{
					T:                   t,
					FailOnUndefinedFunc: true,
					UpdateActivationLabelsAndAnnotationsFunc: func(p *kapiv1.Pod) (*kapiv1.Pod, error) {
						test.GetTestSequence(t, testprefix+"/retryAndKill_retry_fine").PassAtLeastOnce(0)
						return p, nil
					},
				},
			},
			inputPod: func() *kapiv1.Pod {
				p := test.PodGen("A", nil, true, true, "")
				p.Annotations = map[string]string{
					labeling.AnnotationBreakAtKey:    string("1978-12-04T22:11:00+00:00"),
					labeling.AnnotationRetryCountKey: "2",
				}
				return p
			},
			wantErr: false,
		},
		{
			name:     "retryAndPause_retry_fine",
			sequence: test.NewTestSequence(t, testprefix+"/retryAndPause_retry_fine", 1, time.Second),
			fields: fields{
				activatorStrategyConfig: v1.ActivatorStrategy{
					Mode:          v1.ActivatorStrategyModeRetryAndPause,
					MaxRetryCount: v1.NewUInt(3),
				},
				podControl: &test.TestPodControl{
					T:                   t,
					FailOnUndefinedFunc: true,
					UpdateActivationLabelsAndAnnotationsFunc: func(p *kapiv1.Pod) (*kapiv1.Pod, error) {
						test.GetTestSequence(t, testprefix+"/retryAndPause_retry_fine").PassAtLeastOnce(0)
						return p, nil
					},
				},
			},
			inputPod: func() *kapiv1.Pod {
				p := test.PodGen("A", nil, true, true, "")
				p.Annotations = map[string]string{
					labeling.AnnotationBreakAtKey:    string("1978-12-04T22:11:00+00:00"),
					labeling.AnnotationRetryCountKey: "2",
				}
				return p
			},
			wantErr: false,
		},
		{
			name:     "retryAndPause_retry_error",
			sequence: test.NewTestSequence(t, testprefix+"/retryAndPause_retry_error", 1, time.Second),
			fields: fields{
				activatorStrategyConfig: v1.ActivatorStrategy{
					Mode:          v1.ActivatorStrategyModeRetryAndPause,
					MaxRetryCount: v1.NewUInt(3),
				},
				podControl: &test.TestPodControl{
					T:                   t,
					FailOnUndefinedFunc: true,
					UpdateActivationLabelsAndAnnotationsFunc: func(p *kapiv1.Pod) (*kapiv1.Pod, error) {
						test.GetTestSequence(t, testprefix+"/retryAndPause_retry_error").PassAtLeastOnce(0)
						return p, fmt.Errorf("Fake error")
					},
				},
			},
			inputPod: func() *kapiv1.Pod {
				p := test.PodGen("A", nil, true, true, "")
				p.Annotations = map[string]string{
					labeling.AnnotationBreakAtKey:    string("1978-12-04T22:11:00+00:00"),
					labeling.AnnotationRetryCountKey: "2",
				}
				return p
			},
			wantErr: true,
		},
		{
			name:     "retryAndPause_pause_error",
			sequence: test.NewTestSequence(t, testprefix+"/retryAndPause_pause_error", 1, time.Second),
			fields: fields{
				activatorStrategyConfig: v1.ActivatorStrategy{
					Mode:          v1.ActivatorStrategyModeRetryAndPause,
					MaxRetryCount: v1.NewUInt(3),
					MaxPauseCount: v1.NewUInt(1),
				},
				podControl: &test.TestPodControl{
					T:                   t,
					FailOnUndefinedFunc: true,
					UpdatePauseLabelsAndAnnotationsFunc: func(p *kapiv1.Pod) (*kapiv1.Pod, error) {
						test.GetTestSequence(t, testprefix+"/retryAndPause_pause_error").PassAtLeastOnce(0)
						return p, fmt.Errorf("Fake error")
					},
				},
				selector:  labels.SelectorFromSet(map[string]string{"app": "foo"}),
				podLister: test.NewTestPodLister([]*kapiv1.Pod{}),
			},
			inputPod: func() *kapiv1.Pod {
				p := test.PodGen("A", map[string]string{"app": "foo"}, true, true, "")
				p.Annotations = map[string]string{
					labeling.AnnotationBreakAtKey:    string("1978-12-04T22:11:00+00:00"),
					labeling.AnnotationRetryCountKey: "4",
				}
				return p
			},
			wantErr: true,
		},
		{
			name:     "retryAndPause_pause_fine",
			sequence: test.NewTestSequence(t, testprefix+"/retryAndPause_pause_fine", 1, time.Second),
			fields: fields{
				activatorStrategyConfig: v1.ActivatorStrategy{
					Mode:          v1.ActivatorStrategyModeRetryAndPause,
					MaxRetryCount: v1.NewUInt(3),
					MaxPauseCount: v1.NewUInt(1),
				},
				podControl: &test.TestPodControl{
					T:                   t,
					FailOnUndefinedFunc: true,
					UpdatePauseLabelsAndAnnotationsFunc: func(p *kapiv1.Pod) (*kapiv1.Pod, error) {
						test.GetTestSequence(t, testprefix+"/retryAndPause_pause_fine").PassAtLeastOnce(0)
						return p, nil
					},
				},
				selector:  labels.SelectorFromSet(map[string]string{"app": "foo"}),
				podLister: test.NewTestPodLister([]*kapiv1.Pod{}),
			},
			inputPod: func() *kapiv1.Pod {
				p := test.PodGen("A", map[string]string{"app": "foo"}, true, true, "")
				p.Annotations = map[string]string{
					labeling.AnnotationBreakAtKey:    string("1978-12-04T22:11:00+00:00"),
					labeling.AnnotationRetryCountKey: "4",
				}
				return p
			},
			wantErr: false,
		},
		{
			name:     "retryAndPause_kill",
			sequence: test.NewTestSequence(t, testprefix+"/retryAndPause_kill", 1, time.Second),
			fields: fields{
				activatorStrategyConfig: v1.ActivatorStrategy{
					Mode:          v1.ActivatorStrategyModeRetryAndPause,
					MaxRetryCount: v1.NewUInt(3),
					MaxPauseCount: v1.NewUInt(0),
				},
				podControl: &test.TestPodControl{
					T:                   t,
					FailOnUndefinedFunc: true,
					KillPodFunc: func(p *kapiv1.Pod) error {
						test.GetTestSequence(t, testprefix+"/retryAndPause_kill").PassAtLeastOnce(0)
						return nil
					},
				},
				selector:  labels.SelectorFromSet(map[string]string{"app": "foo"}),
				podLister: test.NewTestPodLister([]*kapiv1.Pod{}),
			},
			inputPod: func() *kapiv1.Pod {
				p := test.PodGen("A", map[string]string{"app": "foo"}, true, true, "")
				p.Annotations = map[string]string{
					labeling.AnnotationBreakAtKey:    string("1978-12-04T22:11:00+00:00"),
					labeling.AnnotationRetryCountKey: "4",
				}
				return p
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tpc, ok := tt.fields.podControl.(*test.TestPodControl); ok {
				tpc.Case = tt.name
			}

			b := &ActivatorImpl{
				activatorStrategyConfig: tt.fields.activatorStrategyConfig,
				selector:                tt.fields.selector,
				podLister:               tt.fields.podLister,
				podControl:              tt.fields.podControl,
				breakerName:             tt.fields.breakerName,
				logger:                  tt.fields.logger,
			}

			if err := b.applyActivatorStrategy(tt.inputPod()); (err != nil) != tt.wantErr {
				t.Errorf("ActivatorImpl.applyActivatorStrategy() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.sequence != nil && !tt.sequence.Completed() {
				t.Errorf("the sequence was not completed")
			}

		})
	}

}

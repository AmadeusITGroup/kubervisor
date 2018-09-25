package anomalydetector

import (
	"fmt"
	"reflect"
	"testing"

	"go.uber.org/zap"
	kapiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	kv1 "k8s.io/client-go/listers/core/v1"

	api "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1alpha1"
	"github.com/amadeusitgroup/kubervisor/pkg/labeling"
	test "github.com/amadeusitgroup/kubervisor/test"
)

func TestContinuousValueDeviationAnalyser_GetPodsOutOfBounds(t *testing.T) {
	devlogger, _ := zap.NewDevelopment()

	type fields struct {
		ContinuousValueDeviation api.ContinuousValueDeviation
		selector                 labels.Selector
		analyser                 continuousValueAnalyser
		podLister                kv1.PodNamespaceLister
	}
	tests := []struct {
		name    string
		fields  fields
		want    []*kapiv1.Pod
		wantErr bool
	}{
		{
			name: "analysis error",
			fields: fields{
				ContinuousValueDeviation: *api.DefaultContinuousValueDeviation(&api.ContinuousValueDeviation{}),
				selector:                 nil,
				analyser:                 &testErrorContinuousValueAnalyser{},
				podLister:                test.NewTestPodNamespaceLister(nil, "test-ns"),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "no pod, no error",
			fields: fields{
				ContinuousValueDeviation: *api.DefaultContinuousValueDeviation(&api.ContinuousValueDeviation{}),
				selector:                 labels.Everything(),
				analyser: &testContinuousValueAnalyser{
					deviationByPodName: deviationByPodName{},
				},
				podLister: test.NewTestPodNamespaceLister(nil, "test-ns"),
			},
			want:    []*kapiv1.Pod{},
			wantErr: false,
		},
		{
			name: "bad selector",
			fields: fields{
				ContinuousValueDeviation: *api.DefaultContinuousValueDeviation(&api.ContinuousValueDeviation{}),
				selector:                 labels.Nothing(),
				analyser: &testContinuousValueAnalyser{
					deviationByPodName: deviationByPodName{
						"A": 1.1,
						"B": 1.2,
						"C": 0.2,
					},
				},

				podLister: test.NewTestPodNamespaceLister(
					[]*kapiv1.Pod{
						test.PodGen("A", "test-ns", map[string]string{"app": "foo", "phase": "prd"}, nil, true, true, labeling.LabelTrafficYes),
						test.PodGen("B", "test-ns", map[string]string{"app": "bar", "phase": "prd"}, nil, true, true, labeling.LabelTrafficYes),
						test.PodGen("C", "test-ns", map[string]string{"app": "bar", "phase": "pdt"}, nil, true, true, labeling.LabelTrafficYes),
					}, "test-ns"),
			},
			want:    []*kapiv1.Pod{},
			wantErr: false,
		},
		{
			name: "no traffic label",
			fields: fields{
				ContinuousValueDeviation: *api.DefaultContinuousValueDeviation(&api.ContinuousValueDeviation{}),
				selector:                 labels.Everything(),
				analyser: &testContinuousValueAnalyser{
					deviationByPodName: deviationByPodName{
						"A": 1.1,
						"B": 1.2,
						"C": 0.2,
					},
				},
				podLister: test.NewTestPodNamespaceLister(
					[]*kapiv1.Pod{
						test.PodGen("A", "test-ns", map[string]string{"app": "foo", "phase": "prd"}, nil, true, true, ""),
						test.PodGen("B", "test-ns", map[string]string{"app": "bar", "phase": "prd"}, nil, true, true, ""),
						test.PodGen("C", "test-ns", map[string]string{"app": "bar", "phase": "pdt"}, nil, true, true, ""),
					}, "test-ns"),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "deviation by 70%",
			fields: fields{
				ContinuousValueDeviation: *api.DefaultContinuousValueDeviation(&api.ContinuousValueDeviation{MaxDeviationPercent: api.NewFloat64(50.0)}),
				selector:                 labels.Everything(),
				analyser: &testContinuousValueAnalyser{
					deviationByPodName: deviationByPodName{
						"A": 1.1,
						"B": 1.2,
						"C": 0.2,
					},
				},
				podLister: test.NewTestPodNamespaceLister(
					[]*kapiv1.Pod{
						test.PodGen("A", "test-ns", map[string]string{"app": "foo", "phase": "prd"}, nil, true, true, labeling.LabelTrafficYes),
						test.PodGen("B", "test-ns", map[string]string{"app": "bar", "phase": "prd"}, nil, true, true, labeling.LabelTrafficYes),
						test.PodGen("C", "test-ns", map[string]string{"app": "bar", "phase": "pdt"}, nil, true, true, labeling.LabelTrafficYes)}, "test-ns"),
			},
			want:    []*kapiv1.Pod{test.PodGen("C", "test-ns", map[string]string{"app": "bar", "phase": "pdt"}, nil, true, true, labeling.LabelTrafficYes)},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &ContinuousValueDeviationAnalyser{
				ContinuousValueDeviation: tt.fields.ContinuousValueDeviation,
				selector:                 tt.fields.selector,
				analyser:                 tt.fields.analyser,
				podLister:                tt.fields.podLister,
				logger:                   devlogger,
			}
			got, err := d.GetPodsOutOfBounds()
			if (err != nil) != tt.wantErr {
				t.Errorf("ContinuousValueDeviationAnalyser.GetPodsOutOfBounds() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ContinuousValueDeviationAnalyser.GetPodsOutOfBounds() len[%d] = %v, \n want  len[%d] = %v", len(got), got, len(tt.want), tt.want)
			}
		})
	}
}

type testErrorContinuousValueAnalyser struct{}

func (t *testErrorContinuousValueAnalyser) doAnalysis() (deviationByPodName, error) {
	return nil, fmt.Errorf("error")
}

type testContinuousValueAnalyser struct {
	deviationByPodName
}

func (t *testContinuousValueAnalyser) doAnalysis() (deviationByPodName, error) {
	return t.deviationByPodName, nil
}

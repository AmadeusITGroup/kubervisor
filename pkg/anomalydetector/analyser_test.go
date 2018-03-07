package anomalydetector

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"go.uber.org/zap"
	kapiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	kv1 "k8s.io/client-go/listers/core/v1"

	"github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
	"github.com/amadeusitgroup/podkubervisor/pkg/labeling"

	test "github.com/amadeusitgroup/podkubervisor/test"
)

func TestDiscreteValueOutOfListAnalyser_GetPodsOutOfBounds(t *testing.T) {

	devlogger, _ := zap.NewDevelopment()

	type fields struct {
		DiscreteValueOutOfList v1.DiscreteValueOutOfList
		selector               labels.Selector
		podAnalyser            podAnalyser
		podLister              kv1.PodNamespaceLister
		logger                 *zap.Logger
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
				DiscreteValueOutOfList: *v1.DefaultDiscreteValueOutOfList(&v1.DiscreteValueOutOfList{}),
				selector:               nil,
				podAnalyser:            &testErrorPodAnalyser{},
				podLister:              test.NewTestPodLister(nil, "test-ns"),
				logger:                 devlogger,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "no pod, no error",
			fields: fields{
				DiscreteValueOutOfList: *v1.DefaultDiscreteValueOutOfList(&v1.DiscreteValueOutOfList{}),
				selector:               labels.Everything(), //labels.SelectorFromSet(map[string]string{}),
				podAnalyser:            &testPodAnalyser{okkoByPodName: okkoByPodName{}},
				podLister:              test.NewTestPodLister(nil, "test-ns"),
				logger:                 devlogger,
			},
			want:    []*kapiv1.Pod{},
			wantErr: false,
		},
		{
			name: "bad selector",
			fields: fields{
				DiscreteValueOutOfList: *v1.DefaultDiscreteValueOutOfList(&v1.DiscreteValueOutOfList{}),
				selector:               labels.Nothing(), //labels.SelectorFromSet(map[string]string{}),
				podAnalyser:            &testPodAnalyser{okkoByPodName: okkoByPodName{"A": {10, 0}, "B": {10, 8}, "C": {0, 10}}},
				podLister: test.NewTestPodLister(
					[]*kapiv1.Pod{
						test.PodGen("A", "test-ns", map[string]string{"app": "foo", "phase": "prd"}, true, true, labeling.LabelTrafficYes),
						test.PodGen("B", "test-ns", map[string]string{"app": "bar", "phase": "prd"}, true, true, labeling.LabelTrafficYes),
						test.PodGen("C", "test-ns", map[string]string{"app": "bar", "phase": "pdt"}, true, true, labeling.LabelTrafficYes),
					}, "test-ns"),
				logger: devlogger,
			},
			want:    []*kapiv1.Pod{},
			wantErr: false,
		},
		{
			name: "no traffic label",
			fields: fields{
				DiscreteValueOutOfList: *v1.DefaultDiscreteValueOutOfList(&v1.DiscreteValueOutOfList{}),
				selector:               labels.Everything(), //labels.SelectorFromSet(map[string]string{}),
				podAnalyser:            &testPodAnalyser{okkoByPodName: okkoByPodName{"A": {10, 0}, "B": {10, 8}, "C": {0, 10}}},
				podLister: test.NewTestPodLister(
					[]*kapiv1.Pod{
						test.PodGen("A", "test-ns", map[string]string{"app": "foo", "phase": "prd"}, true, true, ""),
						test.PodGen("B", "test-ns", map[string]string{"app": "bar", "phase": "prd"}, true, true, ""),
						test.PodGen("C", "test-ns", map[string]string{"app": "bar", "phase": "pdt"}, true, true, ""),
					}, "test-ns"),
				logger: devlogger,
			},
			want:    []*kapiv1.Pod{},
			wantErr: false,
		},
		{
			name: "50%",
			fields: fields{
				DiscreteValueOutOfList: *v1.DefaultDiscreteValueOutOfList(&v1.DiscreteValueOutOfList{TolerancePercent: v1.NewUInt(50)}),
				selector:               labels.Everything(), //labels.SelectorFromSet(map[string]string{}),
				podAnalyser:            &testPodAnalyser{okkoByPodName: okkoByPodName{"A": {10, 0}, "B": {10, 8}, "C": {0, 10}}},
				podLister: test.NewTestPodLister(
					[]*kapiv1.Pod{
						test.PodGen("A", "test-ns", map[string]string{"app": "foo", "phase": "prd"}, true, true, labeling.LabelTrafficYes),
						test.PodGen("B", "test-ns", map[string]string{"app": "bar", "phase": "prd"}, true, true, labeling.LabelTrafficYes),
						test.PodGen("C", "test-ns", map[string]string{"app": "bar", "phase": "pdt"}, true, true, labeling.LabelTrafficYes),
					}, "test-ns"),
				logger: devlogger,
			},
			want:    []*kapiv1.Pod{test.PodGen("C", "test-ns", map[string]string{"app": "bar", "phase": "pdt"}, true, true, labeling.LabelTrafficYes)},
			wantErr: false,
		},
		{
			name: "10%",
			fields: fields{
				DiscreteValueOutOfList: *v1.DefaultDiscreteValueOutOfList(&v1.DiscreteValueOutOfList{TolerancePercent: v1.NewUInt(10)}),
				selector:               labels.Everything(), //labels.SelectorFromSet(map[string]string{}),
				podAnalyser:            &testPodAnalyser{okkoByPodName: okkoByPodName{"A": {10, 0}, "B": {10, 8}, "C": {0, 10}}},
				podLister: test.NewTestPodLister(
					[]*kapiv1.Pod{
						test.PodGen("A", "test-ns", map[string]string{"app": "foo", "phase": "prd"}, true, true, labeling.LabelTrafficYes),
						test.PodGen("B", "test-ns", map[string]string{"app": "bar", "phase": "prd"}, true, true, labeling.LabelTrafficYes),
						test.PodGen("C", "test-ns", map[string]string{"app": "bar", "phase": "pdt"}, true, true, labeling.LabelTrafficYes),
					}, "test-ns"),
				logger: devlogger,
			},
			want: []*kapiv1.Pod{
				test.PodGen("B", "test-ns", map[string]string{"app": "bar", "phase": "prd"}, true, true, labeling.LabelTrafficYes),
				test.PodGen("C", "test-ns", map[string]string{"app": "bar", "phase": "pdt"}, true, true, labeling.LabelTrafficYes)},
			wantErr: false,
		},
		{
			name: "10% filter prd",
			fields: fields{
				DiscreteValueOutOfList: *v1.DefaultDiscreteValueOutOfList(&v1.DiscreteValueOutOfList{TolerancePercent: v1.NewUInt(10)}),
				selector:               labels.SelectorFromSet(map[string]string{"phase": "prd"}),
				podAnalyser:            &testPodAnalyser{okkoByPodName: okkoByPodName{"A": {10, 0}, "B": {10, 8}, "C": {0, 10}}},
				podLister: test.NewTestPodLister(
					[]*kapiv1.Pod{
						test.PodGen("A", "test-ns", map[string]string{"app": "foo", "phase": "prd"}, true, true, labeling.LabelTrafficYes),
						test.PodGen("B", "test-ns", map[string]string{"app": "bar", "phase": "prd"}, true, true, labeling.LabelTrafficYes),
						test.PodGen("C", "test-ns", map[string]string{"app": "bar", "phase": "pdt"}, true, true, labeling.LabelTrafficYes),
					}, "test-ns"),
				logger: devlogger,
			},
			want:    []*kapiv1.Pod{test.PodGen("B", "test-ns", map[string]string{"app": "bar", "phase": "prd"}, true, true, labeling.LabelTrafficYes)},
			wantErr: false,
		},
		{
			name: "Not Ready pod C",
			fields: fields{
				DiscreteValueOutOfList: *v1.DefaultDiscreteValueOutOfList(&v1.DiscreteValueOutOfList{TolerancePercent: v1.NewUInt(10)}),
				selector:               labels.Everything(), //labels.SelectorFromSet(map[string]string{}),
				podAnalyser:            &testPodAnalyser{okkoByPodName: okkoByPodName{"A": {10, 0}, "B": {10, 8}, "C": {0, 10}}},
				podLister: test.NewTestPodLister(
					[]*kapiv1.Pod{
						test.PodGen("A", "test-ns", map[string]string{"app": "foo", "phase": "prd"}, true, true, labeling.LabelTrafficYes),
						test.PodGen("B", "test-ns", map[string]string{"app": "bar", "phase": "prd"}, true, true, labeling.LabelTrafficYes),
						test.PodGen("C", "test-ns", map[string]string{"app": "bar", "phase": "pdt"}, true, false, labeling.LabelTrafficYes),
					}, "test-ns"),
				logger: devlogger,
			},
			want: []*kapiv1.Pod{
				test.PodGen("B", "test-ns", map[string]string{"app": "bar", "phase": "prd"}, true, true, labeling.LabelTrafficYes)},
			wantErr: false,
		},
		{
			name: "Not Running pod C",
			fields: fields{
				DiscreteValueOutOfList: *v1.DefaultDiscreteValueOutOfList(&v1.DiscreteValueOutOfList{TolerancePercent: v1.NewUInt(10)}),
				selector:               labels.Everything(), //labels.SelectorFromSet(map[string]string{}),
				podAnalyser:            &testPodAnalyser{okkoByPodName: okkoByPodName{"A": {10, 0}, "B": {10, 8}, "C": {0, 10}}},
				podLister: test.NewTestPodLister(
					[]*kapiv1.Pod{
						test.PodGen("A", "test-ns", map[string]string{"app": "foo", "phase": "prd"}, true, true, labeling.LabelTrafficYes),
						test.PodGen("B", "test-ns", map[string]string{"app": "bar", "phase": "prd"}, true, true, labeling.LabelTrafficYes),
						test.PodGen("C", "test-ns", map[string]string{"app": "bar", "phase": "pdt"}, true, false, labeling.LabelTrafficYes),
					}, "test-ns"),
				logger: devlogger,
			},
			want: []*kapiv1.Pod{
				test.PodGen("B", "test-ns", map[string]string{"app": "bar", "phase": "prd"}, true, true, labeling.LabelTrafficYes)},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		devlogger.Sugar().Infof("Running test %s", tt.name)
		t.Run(tt.name, func(t *testing.T) {
			d := &DiscreteValueOutOfListAnalyser{
				DiscreteValueOutOfList: tt.fields.DiscreteValueOutOfList,
				selector:               tt.fields.selector,
				podAnalyser:            tt.fields.podAnalyser,
				podLister:              tt.fields.podLister,
				logger:                 tt.fields.logger,
			}
			got, err := d.GetPodsOutOfBounds()
			if (err != nil) != tt.wantErr {
				t.Errorf("DiscreteValueOutOfListAnalyser.GetPodsOutOfBounds() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("Got DiscreteValueOutOfListAnalyser.GetPodsOutOfBounds() = %v,\n want %v", got, tt.want)
				return
			}

			sort.SliceStable(got, func(i, j int) bool { return got[i].Name < got[j].Name })
			sort.SliceStable(got, func(i, j int) bool { return tt.want[i].Name < tt.want[j].Name })

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Got DiscreteValueOutOfListAnalyser.GetPodsOutOfBounds() = %v,\n want %v", got, tt.want)
			}
		})
	}

}

type testErrorPodAnalyser struct{}

func (t *testErrorPodAnalyser) doAnalysis() (okkoByPodName, error) { return nil, fmt.Errorf("error") }

type testPodAnalyser struct {
	okkoByPodName
}

func (t *testPodAnalyser) doAnalysis() (okkoByPodName, error) { return t.okkoByPodName, nil }

type testPodLister struct {
}

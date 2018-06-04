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

	api "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1alpha1"
	"github.com/amadeusitgroup/kubervisor/pkg/labeling"

	test "github.com/amadeusitgroup/kubervisor/test"
)

func TestDiscreteValueOutOfListAnalyser_GetPodsOutOfBounds(t *testing.T) {

	devlogger, _ := zap.NewDevelopment()

	type fields struct {
		DiscreteValueOutOfList api.DiscreteValueOutOfList
		selector               labels.Selector
		analyser               discreteValueAnalyser
		podLister              kv1.PodNamespaceLister
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
				DiscreteValueOutOfList: *api.DefaultDiscreteValueOutOfList(&api.DiscreteValueOutOfList{}),
				selector:               nil,
				analyser:               &testErrorDiscreateValueAnalyser{},
				podLister:              test.NewTestPodNamespaceLister(nil, "test-ns"),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "no pod, no error",
			fields: fields{
				DiscreteValueOutOfList: *api.DefaultDiscreteValueOutOfList(&api.DiscreteValueOutOfList{}),
				selector:               labels.Everything(),
				analyser:               &testDiscreateValueAnalyser{okkoByPodName: okkoByPodName{}},
				podLister:              test.NewTestPodNamespaceLister(nil, "test-ns"),
			},
			want:    []*kapiv1.Pod{},
			wantErr: false,
		},
		{
			name: "bad selector",
			fields: fields{
				DiscreteValueOutOfList: *api.DefaultDiscreteValueOutOfList(&api.DiscreteValueOutOfList{}),
				selector:               labels.Nothing(),
				analyser:               &testDiscreateValueAnalyser{okkoByPodName: okkoByPodName{"A": {10, 0}, "B": {10, 8}, "C": {0, 10}}},
				podLister: test.NewTestPodNamespaceLister(
					[]*kapiv1.Pod{
						test.PodGen("A", "test-ns", map[string]string{"app": "foo", "phase": "prd"}, true, true, labeling.LabelTrafficYes),
						test.PodGen("B", "test-ns", map[string]string{"app": "bar", "phase": "prd"}, true, true, labeling.LabelTrafficYes),
						test.PodGen("C", "test-ns", map[string]string{"app": "bar", "phase": "pdt"}, true, true, labeling.LabelTrafficYes),
					}, "test-ns"),
			},
			want:    []*kapiv1.Pod{},
			wantErr: false,
		},
		{
			name: "no traffic label",
			fields: fields{
				DiscreteValueOutOfList: *api.DefaultDiscreteValueOutOfList(&api.DiscreteValueOutOfList{}),
				selector:               labels.Everything(),
				analyser:               &testDiscreateValueAnalyser{okkoByPodName: okkoByPodName{"A": {10, 0}, "B": {10, 8}, "C": {0, 10}}},
				podLister: test.NewTestPodNamespaceLister(
					[]*kapiv1.Pod{
						test.PodGen("A", "test-ns", map[string]string{"app": "foo", "phase": "prd"}, true, true, ""),
						test.PodGen("B", "test-ns", map[string]string{"app": "bar", "phase": "prd"}, true, true, ""),
						test.PodGen("C", "test-ns", map[string]string{"app": "bar", "phase": "pdt"}, true, true, ""),
					}, "test-ns"),
			},
			want:    []*kapiv1.Pod{},
			wantErr: false,
		},
		{
			name: "50%",
			fields: fields{
				DiscreteValueOutOfList: *api.DefaultDiscreteValueOutOfList(&api.DiscreteValueOutOfList{TolerancePercent: api.NewUInt(50)}),
				selector:               labels.Everything(),
				analyser:               &testDiscreateValueAnalyser{okkoByPodName: okkoByPodName{"A": {10, 0}, "B": {10, 8}, "C": {0, 10}}},
				podLister: test.NewTestPodNamespaceLister(
					[]*kapiv1.Pod{
						test.PodGen("A", "test-ns", map[string]string{"app": "foo", "phase": "prd"}, true, true, labeling.LabelTrafficYes),
						test.PodGen("B", "test-ns", map[string]string{"app": "bar", "phase": "prd"}, true, true, labeling.LabelTrafficYes),
						test.PodGen("C", "test-ns", map[string]string{"app": "bar", "phase": "pdt"}, true, true, labeling.LabelTrafficYes),
					}, "test-ns"),
			},
			want:    []*kapiv1.Pod{test.PodGen("C", "test-ns", map[string]string{"app": "bar", "phase": "pdt"}, true, true, labeling.LabelTrafficYes)},
			wantErr: false,
		},
		{
			name: "10%",
			fields: fields{
				DiscreteValueOutOfList: *api.DefaultDiscreteValueOutOfList(&api.DiscreteValueOutOfList{TolerancePercent: api.NewUInt(10)}),
				selector:               labels.Everything(),
				analyser:               &testDiscreateValueAnalyser{okkoByPodName: okkoByPodName{"A": {10, 0}, "B": {10, 8}, "C": {0, 10}}},
				podLister: test.NewTestPodNamespaceLister(
					[]*kapiv1.Pod{
						test.PodGen("A", "test-ns", map[string]string{"app": "foo", "phase": "prd"}, true, true, labeling.LabelTrafficYes),
						test.PodGen("B", "test-ns", map[string]string{"app": "bar", "phase": "prd"}, true, true, labeling.LabelTrafficYes),
						test.PodGen("C", "test-ns", map[string]string{"app": "bar", "phase": "pdt"}, true, true, labeling.LabelTrafficYes),
					}, "test-ns"),
			},
			want: []*kapiv1.Pod{
				test.PodGen("B", "test-ns", map[string]string{"app": "bar", "phase": "prd"}, true, true, labeling.LabelTrafficYes),
				test.PodGen("C", "test-ns", map[string]string{"app": "bar", "phase": "pdt"}, true, true, labeling.LabelTrafficYes)},
			wantErr: false,
		},
		{
			name: "10% filter prd",
			fields: fields{
				DiscreteValueOutOfList: *api.DefaultDiscreteValueOutOfList(&api.DiscreteValueOutOfList{TolerancePercent: api.NewUInt(10)}),
				selector:               labels.SelectorFromSet(map[string]string{"phase": "prd"}),
				analyser:               &testDiscreateValueAnalyser{okkoByPodName: okkoByPodName{"A": {10, 0}, "B": {10, 8}, "C": {0, 10}}},
				podLister: test.NewTestPodNamespaceLister(
					[]*kapiv1.Pod{
						test.PodGen("A", "test-ns", map[string]string{"app": "foo", "phase": "prd"}, true, true, labeling.LabelTrafficYes),
						test.PodGen("B", "test-ns", map[string]string{"app": "bar", "phase": "prd"}, true, true, labeling.LabelTrafficYes),
						test.PodGen("C", "test-ns", map[string]string{"app": "bar", "phase": "pdt"}, true, true, labeling.LabelTrafficYes),
					}, "test-ns"),
			},
			want:    []*kapiv1.Pod{test.PodGen("B", "test-ns", map[string]string{"app": "bar", "phase": "prd"}, true, true, labeling.LabelTrafficYes)},
			wantErr: false,
		},
		{
			name: "Not Ready pod C",
			fields: fields{
				DiscreteValueOutOfList: *api.DefaultDiscreteValueOutOfList(&api.DiscreteValueOutOfList{TolerancePercent: api.NewUInt(10)}),
				selector:               labels.Everything(),
				analyser:               &testDiscreateValueAnalyser{okkoByPodName: okkoByPodName{"A": {10, 0}, "B": {10, 8}, "C": {0, 10}}},
				podLister: test.NewTestPodNamespaceLister(
					[]*kapiv1.Pod{
						test.PodGen("A", "test-ns", map[string]string{"app": "foo", "phase": "prd"}, true, true, labeling.LabelTrafficYes),
						test.PodGen("B", "test-ns", map[string]string{"app": "bar", "phase": "prd"}, true, true, labeling.LabelTrafficYes),
						test.PodGen("C", "test-ns", map[string]string{"app": "bar", "phase": "pdt"}, true, false, labeling.LabelTrafficYes),
					}, "test-ns"),
			},
			want: []*kapiv1.Pod{
				test.PodGen("B", "test-ns", map[string]string{"app": "bar", "phase": "prd"}, true, true, labeling.LabelTrafficYes)},
			wantErr: false,
		},
		{
			name: "Not Running pod C",
			fields: fields{
				DiscreteValueOutOfList: *api.DefaultDiscreteValueOutOfList(&api.DiscreteValueOutOfList{TolerancePercent: api.NewUInt(10)}),
				selector:               labels.Everything(),
				analyser:               &testDiscreateValueAnalyser{okkoByPodName: okkoByPodName{"A": {10, 0}, "B": {10, 8}, "C": {0, 10}}},
				podLister: test.NewTestPodNamespaceLister(
					[]*kapiv1.Pod{
						test.PodGen("A", "test-ns", map[string]string{"app": "foo", "phase": "prd"}, true, true, labeling.LabelTrafficYes),
						test.PodGen("B", "test-ns", map[string]string{"app": "bar", "phase": "prd"}, true, true, labeling.LabelTrafficYes),
						test.PodGen("C", "test-ns", map[string]string{"app": "bar", "phase": "pdt"}, true, false, labeling.LabelTrafficYes),
					}, "test-ns"),
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
				analyser:               tt.fields.analyser,
				podLister:              tt.fields.podLister,
				logger:                 devlogger,
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

type testErrorDiscreateValueAnalyser struct{}

func (t *testErrorDiscreateValueAnalyser) doAnalysis() (okkoByPodName, error) {
	return nil, fmt.Errorf("error")
}

type testDiscreateValueAnalyser struct {
	okkoByPodName
}

func (t *testDiscreateValueAnalyser) doAnalysis() (okkoByPodName, error) {
	return t.okkoByPodName, nil
}

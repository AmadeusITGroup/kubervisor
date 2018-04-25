package pod

import (
	"reflect"
	"testing"

	"github.com/amadeusitgroup/kubervisor/pkg/labeling"
	test "github.com/amadeusitgroup/kubervisor/test"
	kapiv1 "k8s.io/api/core/v1"
)

func TestPurgeNotReadyPods(t *testing.T) {
	type args struct {
		pods []*kapiv1.Pod
	}
	tests := []struct {
		name string
		args args
		want []*kapiv1.Pod
	}{
		{
			name: "empty",
			want: []*kapiv1.Pod{},
		},
		{
			name: "all ok",
			args: args{
				pods: []*kapiv1.Pod{
					test.PodGen("A", "test-ns", nil, true, true, ""),
					test.PodGen("B", "test-ns", nil, true, true, ""),
					test.PodGen("C", "test-ns", nil, true, true, ""),
				},
			},
			want: []*kapiv1.Pod{
				test.PodGen("A", "test-ns", nil, true, true, ""),
				test.PodGen("B", "test-ns", nil, true, true, ""),
				test.PodGen("C", "test-ns", nil, true, true, ""),
			},
		},
		{
			name: "mix",
			args: args{
				pods: []*kapiv1.Pod{
					test.PodGen("A", "test-ns", nil, false, true, ""),
					test.PodGen("B", "test-ns", nil, true, false, ""),
					test.PodGen("C", "test-ns", nil, false, false, ""),
					test.PodGen("D", "test-ns", nil, true, true, ""),
				},
			},
			want: []*kapiv1.Pod{
				test.PodGen("D", "test-ns", nil, true, true, ""),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PurgeNotReadyPods(tt.args.pods); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PurgeNotReadyPods() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKeepRunningPods(t *testing.T) {
	type args struct {
		pods []*kapiv1.Pod
	}
	tests := []struct {
		name string
		args args
		want []*kapiv1.Pod
	}{
		{
			name: "empty",
			want: []*kapiv1.Pod{},
		},
		{
			name: "all ok",
			args: args{
				pods: []*kapiv1.Pod{
					test.PodGen("A", "test-ns", nil, true, true, ""),
					test.PodGen("B", "test-ns", nil, true, true, ""),
				},
			},
			want: []*kapiv1.Pod{
				test.PodGen("A", "test-ns", nil, true, true, ""),
				test.PodGen("B", "test-ns", nil, true, true, ""),
			},
		},
		{
			name: "mix",
			args: args{
				pods: []*kapiv1.Pod{
					test.PodGen("A", "test-ns", nil, false, true, ""),
					test.PodGen("B", "test-ns", nil, true, false, ""),
				},
			},
			want: []*kapiv1.Pod{
				test.PodGen("B", "test-ns", nil, true, false, ""),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := KeepRunningPods(tt.args.pods); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("KeepRunningPods() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKeepWithTrafficYesPods(t *testing.T) {
	type args struct {
		pods []*kapiv1.Pod
	}
	tests := []struct {
		name string
		args args
		want []*kapiv1.Pod
	}{
		{
			name: "empty",
			want: []*kapiv1.Pod{},
		},
		{
			name: "all ok",
			args: args{
				pods: []*kapiv1.Pod{
					test.PodGen("A", "test-ns", nil, true, true, labeling.LabelTrafficYes),
					test.PodGen("B", "test-ns", nil, true, true, labeling.LabelTrafficYes),
				},
			},
			want: []*kapiv1.Pod{
				test.PodGen("A", "test-ns", nil, true, true, labeling.LabelTrafficYes),
				test.PodGen("B", "test-ns", nil, true, true, labeling.LabelTrafficYes),
			},
		},
		{
			name: "mix",
			args: args{
				pods: []*kapiv1.Pod{
					test.PodGen("A", "test-ns", nil, false, true, labeling.LabelTrafficNo),
					test.PodGen("B", "test-ns", nil, true, false, labeling.LabelTrafficYes),
					test.PodGen("C", "test-ns", nil, false, true, ""),
				},
			},
			want: []*kapiv1.Pod{
				test.PodGen("B", "test-ns", nil, true, false, labeling.LabelTrafficYes),
			},
		},
		{
			name: "none",
			args: args{
				pods: []*kapiv1.Pod{
					test.PodGen("A", "test-ns", nil, false, true, labeling.LabelTrafficNo),
					test.PodGen("B", "test-ns", nil, true, false, labeling.LabelTrafficPause),
				},
			},
			want: []*kapiv1.Pod{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := KeepWithTrafficYesPods(tt.args.pods); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("KeepWithTrafficYesPods() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExcludeFromSlice(t *testing.T) {
	pod1 := test.PodGen("pod1", "test-ns", nil, true, false, labeling.LabelTrafficYes)
	pod2 := test.PodGen("pod2", "test-ns", nil, true, false, labeling.LabelTrafficYes)
	pod3 := test.PodGen("pod3", "test-ns", nil, true, false, labeling.LabelTrafficYes)
	pod4 := test.PodGen("pod4", "test-ns", nil, true, false, labeling.LabelTrafficYes)
	pod5 := test.PodGen("pod5", "test-ns", nil, true, false, labeling.LabelTrafficYes)

	type args struct {
		fromSlice []*kapiv1.Pod
		inSlice   []*kapiv1.Pod
	}
	tests := []struct {
		name string
		args args
		want []*kapiv1.Pod
	}{
		{
			name: "similar slice",
			args: args{
				fromSlice: []*kapiv1.Pod{pod1, pod2, pod3},
				inSlice:   []*kapiv1.Pod{pod1, pod2, pod3},
			},
			want: []*kapiv1.Pod{},
		},
		{
			name: "missing pods",
			args: args{
				fromSlice: []*kapiv1.Pod{pod1, pod2, pod3, pod4, pod5},
				inSlice:   []*kapiv1.Pod{pod1, pod2, pod3},
			},
			want: []*kapiv1.Pod{pod4, pod5},
		},
		{
			name: "additional pods",
			args: args{
				fromSlice: []*kapiv1.Pod{pod1, pod2, pod3, pod4, pod5},
				inSlice:   []*kapiv1.Pod{pod1, pod2, pod3, pod4, pod5},
			},
			want: []*kapiv1.Pod{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExcludeFromSlice(tt.args.fromSlice, tt.args.inSlice); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExcludeFromSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

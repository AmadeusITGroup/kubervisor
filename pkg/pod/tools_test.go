package pod

import (
	"reflect"
	"testing"

	kapiv1 "k8s.io/api/core/v1"

	"github.com/amadeusitgroup/podkubervisor/pkg/labeling"

	test "github.com/amadeusitgroup/podkubervisor/test"
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

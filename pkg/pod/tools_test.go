package pod

import (
	"reflect"
	"testing"

	"github.com/amadeusitgroup/podkubervisor/pkg/labeling"
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
					podGen("A", nil, true, true, ""),
					podGen("B", nil, true, true, ""),
					podGen("C", nil, true, true, ""),
				},
			},
			want: []*kapiv1.Pod{
				podGen("A", nil, true, true, ""),
				podGen("B", nil, true, true, ""),
				podGen("C", nil, true, true, ""),
			},
		},
		{
			name: "mix",
			args: args{
				pods: []*kapiv1.Pod{
					podGen("A", nil, false, true, ""),
					podGen("B", nil, true, false, ""),
					podGen("C", nil, false, false, ""),
					podGen("D", nil, true, true, ""),
				},
			},
			want: []*kapiv1.Pod{
				podGen("D", nil, true, true, ""),
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
					podGen("A", nil, true, true, ""),
					podGen("B", nil, true, true, ""),
				},
			},
			want: []*kapiv1.Pod{
				podGen("A", nil, true, true, ""),
				podGen("B", nil, true, true, ""),
			},
		},
		{
			name: "mix",
			args: args{
				pods: []*kapiv1.Pod{
					podGen("A", nil, false, true, ""),
					podGen("B", nil, true, false, ""),
				},
			},
			want: []*kapiv1.Pod{
				podGen("B", nil, true, false, ""),
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
					podGen("A", nil, true, true, labeling.LabelTrafficYes),
					podGen("B", nil, true, true, labeling.LabelTrafficYes),
				},
			},
			want: []*kapiv1.Pod{
				podGen("A", nil, true, true, labeling.LabelTrafficYes),
				podGen("B", nil, true, true, labeling.LabelTrafficYes),
			},
		},
		{
			name: "mix",
			args: args{
				pods: []*kapiv1.Pod{
					podGen("A", nil, false, true, labeling.LabelTrafficNo),
					podGen("B", nil, true, false, labeling.LabelTrafficYes),
					podGen("C", nil, false, true, ""),
				},
			},
			want: []*kapiv1.Pod{
				podGen("B", nil, true, false, labeling.LabelTrafficYes),
			},
		},
		{
			name: "none",
			args: args{
				pods: []*kapiv1.Pod{
					podGen("A", nil, false, true, labeling.LabelTrafficNo),
					podGen("B", nil, true, false, labeling.LabelTrafficPause),
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

func podGen(name string, labels map[string]string, running, ready bool, trafficLabel labeling.LabelTraffic) *kapiv1.Pod {
	p := kapiv1.Pod{}
	p.Name = name
	if trafficLabel != "" {
		if labels == nil {
			labels = map[string]string{}
		}
		p.SetLabels(labels)
		labeling.SetTraficLabel(&p, trafficLabel)
	}
	if running {
		p.Status = kapiv1.PodStatus{Phase: kapiv1.PodRunning}
		if ready {
			p.Status.Conditions = []kapiv1.PodCondition{{Type: kapiv1.PodReady, Status: kapiv1.ConditionTrue}}
		} else {
			p.Status.Conditions = []kapiv1.PodCondition{{Type: kapiv1.PodReady, Status: kapiv1.ConditionFalse}}
		}
	} else {
		p.Status = kapiv1.PodStatus{Phase: kapiv1.PodUnknown}
	}
	return &p
}

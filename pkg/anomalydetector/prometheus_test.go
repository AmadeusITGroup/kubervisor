package anomalydetector

import (
	"reflect"
	"testing"

	promClient "github.com/prometheus/client_golang/api"
	"github.com/prometheus/common/model"
	"go.uber.org/zap"

	"github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
)

func Test_promDiscreteValueOutOfListAnalyser_buildCounters(t *testing.T) {
	type fields struct {
		config           v1.DiscreteValueOutOfList
		prometheusClient promClient.Client
		logger           *zap.Logger
		valueCheckerFunc func(value string) (ok bool)
	}
	type args struct {
		vector model.Vector
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   okkoByPodName
	}{
		{
			name: "empty",
			fields: fields{
				config:           v1.DiscreteValueOutOfList{PodNameKey: "podname", Key: "code"},
				valueCheckerFunc: func(value string) bool { return ContainsString([]string{"200"}, value) },
			},
			args: args{
				vector: model.Vector{},
			},
			want: okkoByPodName{},
		},
		{
			name: "one ok element; inclusion",
			fields: fields{
				config: *v1.DefaultDiscreteValueOutOfList(
					&v1.DiscreteValueOutOfList{PodNameKey: "podname", Key: "code"}),
				valueCheckerFunc: func(value string) bool { return ContainsString([]string{"200"}, value) },
			},
			args: args{
				vector: model.Vector{
					&model.Sample{
						Metric: model.Metric{"code": "200", "podname": "david"},
						Value:  1.0,
					},
				},
			},
			want: okkoByPodName{"david": {1.0, 0.0}},
		},
		{
			name: "one ko element; inclusion",
			fields: fields{
				config: *v1.DefaultDiscreteValueOutOfList(
					&v1.DiscreteValueOutOfList{PodNameKey: "podname", Key: "code"}),
				valueCheckerFunc: func(value string) bool { return ContainsString([]string{"200"}, value) },
			},
			args: args{
				vector: model.Vector{
					&model.Sample{
						Metric: model.Metric{"code": "500", "podname": "david"},
						Value:  1.0,
					},
				},
			},
			want: okkoByPodName{"david": {0.0, 1.0}},
		},
		{
			name: "one ok element; exclusion",
			fields: fields{
				config: *v1.DefaultDiscreteValueOutOfList(
					&v1.DiscreteValueOutOfList{PodNameKey: "podname", Key: "code"}),
				valueCheckerFunc: func(value string) bool { return !ContainsString([]string{"500"}, value) },
			},
			args: args{
				vector: model.Vector{
					&model.Sample{
						Metric: model.Metric{"code": "200", "podname": "david"},
						Value:  1.0,
					},
				},
			},
			want: okkoByPodName{"david": {1.0, 0.0}},
		},
		{
			name: "one ko element; exclusion",
			fields: fields{
				config:           v1.DiscreteValueOutOfList{PodNameKey: "podname", Key: "code"},
				valueCheckerFunc: func(value string) bool { return !ContainsString([]string{"500"}, value) },
			},
			args: args{
				vector: model.Vector{
					&model.Sample{
						Metric: model.Metric{"code": "500", "podname": "david"},
						Value:  1.0,
					},
				},
			},
			want: okkoByPodName{"david": {0.0, 1.0}},
		},
		{
			name: "complex; inclusion",
			fields: fields{
				config:           v1.DiscreteValueOutOfList{PodNameKey: "podname", Key: "code"},
				valueCheckerFunc: func(value string) bool { return ContainsString([]string{"200"}, value) },
			},
			args: args{
				vector: model.Vector{
					&model.Sample{
						Metric: model.Metric{"code": "200", "podname": "david"},
						Value:  10.0,
					},
					&model.Sample{
						Metric: model.Metric{"code": "200", "podname": "cedric"},
						Value:  20.0,
					},
					&model.Sample{
						Metric: model.Metric{"code": "500", "podname": "david"},
						Value:  3.0,
					},
					&model.Sample{
						Metric: model.Metric{"code": "404", "podname": "david"},
						Value:  6.0,
					},
					&model.Sample{
						Metric: model.Metric{"code": "500", "podname": "cedric"},
						Value:  8.0,
					},
					&model.Sample{
						Metric: model.Metric{"code": "200", "podname": "dario"},
						Value:  30.0,
					},
				},
			},
			want: okkoByPodName{"david": {10.0, 9.0}, "cedric": {20.0, 8.0}, "dario": {30.0, 0.0}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &promDiscreteValueOutOfListAnalyser{
				config:           tt.fields.config,
				prometheusClient: tt.fields.prometheusClient,
				logger:           tt.fields.logger,
				valueCheckerFunc: tt.fields.valueCheckerFunc,
			}
			if got := p.buildCounters(tt.args.vector); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("promDiscreteValueOutOfListAnalyser.buildCounters() = %v, want %v", got, tt.want)
			}
		})
	}
}

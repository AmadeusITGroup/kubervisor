package anomalydetector

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1"
	promApi "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"go.uber.org/zap"
)

func Test_promDiscreteValueOutOfListAnalyser_buildCounters(t *testing.T) {
	type fields struct {
		config           v1.DiscreteValueOutOfList
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
				logger:           tt.fields.logger,
				valueCheckerFunc: tt.fields.valueCheckerFunc,
			}
			if got := p.buildCounters(tt.args.vector); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("promDiscreteValueOutOfListAnalyser.buildCounters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_promDiscreteValueOutOfListAnalyser_doAnalysis(t *testing.T) {
	type fields struct {
		config           v1.DiscreteValueOutOfList
		qAPI             promApi.API
		logger           *zap.Logger
		valueCheckerFunc func(value string) (ok bool)
	}
	tests := []struct {
		name    string
		fields  fields
		want    okkoByPodName
		wantErr bool
	}{
		{
			name: "caseErrorQuery",
			fields: fields{
				config: v1.DiscreteValueOutOfList{},
				qAPI: &testPrometheusAPI{
					err: fmt.Errorf("A prom Error"),
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "okEmpty",
			fields: fields{
				config: v1.DiscreteValueOutOfList{
					PodNameKey: "pod",
					Key:        "code",
				},
				qAPI: &testPrometheusAPI{
					err:   nil,
					value: model.Vector([]*model.Sample{}),
				},
			},
			want:    okkoByPodName{},
			wantErr: false,
		},
		{
			name: "badCast",
			fields: fields{
				config: v1.DiscreteValueOutOfList{
					PodNameKey: "pod",
				},
				qAPI: &testPrometheusAPI{
					err:   nil,
					value: nil,
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &promDiscreteValueOutOfListAnalyser{
				config:           tt.fields.config,
				queyrAPI:         tt.fields.qAPI,
				logger:           tt.fields.logger,
				valueCheckerFunc: tt.fields.valueCheckerFunc,
			}
			got, err := p.doAnalysis()
			if (err != nil) != tt.wantErr {
				t.Errorf("promDiscreteValueOutOfListAnalyser.doAnalysis() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("promDiscreteValueOutOfListAnalyser.doAnalysis() = %v, want %v", got, tt.want)
			}
		})
	}
}

type testPrometheusAPI struct {
	value  model.Value
	lvalue model.LabelValues
	err    error
}

// Query performs a query for the given time.
func (tAPI *testPrometheusAPI) Query(ctx context.Context, query string, ts time.Time) (model.Value, error) {
	return tAPI.value, tAPI.err
}

// QueryRange performs a query for the given range.
func (tAPI *testPrometheusAPI) QueryRange(ctx context.Context, query string, r promApi.Range) (model.Value, error) {
	return tAPI.value, tAPI.err
}

// LabelValues performs a query for the values of the given label.
func (tAPI *testPrometheusAPI) LabelValues(ctx context.Context, label string) (model.LabelValues, error) {
	return tAPI.lvalue, tAPI.err
}

func Test_promContinuousValueDeviationAnalyser_doAnalysis(t *testing.T) {
	type fields struct {
		config v1.ContinuousValueDeviation
		qAPI   promApi.API
		logger *zap.Logger
	}
	tests := []struct {
		name    string
		fields  fields
		want    deviationByPodName
		wantErr bool
	}{
		{
			name: "caseErrorQuery",
			fields: fields{
				config: v1.ContinuousValueDeviation{},
				qAPI: &testPrometheusAPI{
					err: fmt.Errorf("A prom Error"),
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "podA",
			fields: fields{
				config: v1.ContinuousValueDeviation{
					PodNameKey: "pod",
				},
				qAPI: &testPrometheusAPI{
					err: nil,
					value: model.Vector([]*model.Sample{
						&model.Sample{
							Metric: model.Metric(model.LabelSet(map[model.LabelName]model.LabelValue{"pod": "podA"})),
							Value:  model.SampleValue(42.0),
						},
					}),
				},
			},
			want:    map[string]float64{"podA": 42.0},
			wantErr: false,
		},
		{
			name: "badCast",
			fields: fields{
				config: v1.ContinuousValueDeviation{
					PodNameKey: "pod",
				},
				qAPI: &testPrometheusAPI{
					err:   nil,
					value: nil,
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &promContinuousValueDeviationAnalyser{
				config:   tt.fields.config,
				queryAPI: tt.fields.qAPI,
				logger:   tt.fields.logger,
			}
			got, err := p.doAnalysis()
			if (err != nil) != tt.wantErr {
				t.Errorf("promContinuousValueDeviationAnalyser.doAnalysis() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("promContinuousValueDeviationAnalyser.doAnalysis() = %v, want %v", got, tt.want)
			}
		})
	}
}

package anomalydetector

import (
	"reflect"
	"testing"

	"go.uber.org/zap"
	kapiv1 "k8s.io/api/core/v1"

	api "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1alpha1"
)

type emptyCustomAnomalyDetectorT struct {
}

func (e *emptyCustomAnomalyDetectorT) GetPodsOutOfBounds() ([]*kapiv1.Pod, error) { return nil, nil }

var emptyCustomAnomalyDetector AnomalyDetector = &emptyCustomAnomalyDetectorT{}

func customFactory(cfg FactoryConfig) (AnomalyDetector, error) { return emptyCustomAnomalyDetector, nil }

func TestNew(t *testing.T) {

	devLogger, _ := zap.NewDevelopment()

	type args struct {
		cfg FactoryConfig
	}
	tests := []struct {
		name         string
		args         args
		want         AnomalyDetector
		wantErr      bool
		compareValue bool
	}{
		{
			name: "error",
			args: args{
				cfg: FactoryConfig{},
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "custom",
			args: args{
				cfg: FactoryConfig{customFactory: func(cfg FactoryConfig) (AnomalyDetector, error) { return emptyCustomAnomalyDetector, nil }},
			},
			wantErr:      false,
			want:         emptyCustomAnomalyDetector,
			compareValue: true,
		},
		{
			name: "missing Param",
			args: args{
				cfg: FactoryConfig{
					Config: Config{
						Logger:    devLogger,
						PodLister: nil,
						BreakerStrategyConfig: api.BreakerStrategy{
							DiscreteValueOutOfList: &api.DiscreteValueOutOfList{},
						},
					},
				},
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "continuousValueDeviation",
			args: args{
				cfg: FactoryConfig{
					Config: Config{
						Logger:    devLogger,
						PodLister: nil,
						BreakerStrategyConfig: api.BreakerStrategy{
							ContinuousValueDeviation: &api.ContinuousValueDeviation{
								MaxDeviationPercent: api.NewFloat64(30.0),
								PodNameKey:          "pod",
								PrometheusService:   "PrometheusService",
								PromQL:              "fakeQuery",
							},
						},
					},
				},
			},
			wantErr: false,
			want:    nil,
		},
		{
			name: "continuousValueDeviation_ErrorNoQuery",
			args: args{
				cfg: FactoryConfig{
					Config: Config{
						Logger:    devLogger,
						PodLister: nil,
						BreakerStrategyConfig: api.BreakerStrategy{
							ContinuousValueDeviation: &api.ContinuousValueDeviation{
								MaxDeviationPercent: api.NewFloat64(30.0),
								PodNameKey:          "pod",
								PrometheusService:   "PrometheusService",
							},
						},
					},
				},
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "continuousValueDeviation_ValidationError",
			args: args{
				cfg: FactoryConfig{
					Config: Config{
						Logger:    devLogger,
						PodLister: nil,
						BreakerStrategyConfig: api.BreakerStrategy{
							ContinuousValueDeviation: &api.ContinuousValueDeviation{},
						},
					},
				},
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "good value only",
			args: args{
				cfg: FactoryConfig{
					Config: Config{
						Logger:    devLogger,
						PodLister: nil,
						BreakerStrategyConfig: api.BreakerStrategy{
							DiscreteValueOutOfList: &api.DiscreteValueOutOfList{
								PromQL:            "fake query",
								PrometheusService: "PrometheusService",
								GoodValues:        []string{"1"},
								Key:               "code",
								PodNameKey:        "podname",
							},
						},
					},
				},
			},
			wantErr: false,
			want:    nil,
		},
		{
			name: "bad value only",
			args: args{
				cfg: FactoryConfig{
					Config: Config{
						Logger:    devLogger,
						PodLister: nil,
						BreakerStrategyConfig: api.BreakerStrategy{
							DiscreteValueOutOfList: &api.DiscreteValueOutOfList{
								PromQL:            "fake query",
								PrometheusService: "PrometheusService",
								BadValues:         []string{"0"},
								Key:               "code",
								PodNameKey:        "podname",
							},
						},
					},
				},
			},
			wantErr: false,
			want:    nil,
		},
		{
			name: "customService",
			args: args{
				cfg: FactoryConfig{
					Config: Config{
						Logger:    devLogger,
						PodLister: nil,
						BreakerStrategyConfig: api.BreakerStrategy{
							CustomService: "CustomURI",
						},
					},
				},
			},
			wantErr: false,
			want:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				return
			}

			if tt.compareValue && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

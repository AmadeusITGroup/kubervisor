package anomalydetector

import (
	"reflect"
	"testing"

	"go.uber.org/zap"
	kapiv1 "k8s.io/api/core/v1"

	"github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
)

type emptyCustomAnomalyDetectorT struct {
}

func (e *emptyCustomAnomalyDetectorT) GetPodsOutOfBounds() ([]*kapiv1.Pod, error) { return nil, nil }

var emptyCustomAnomalyDetector AnomalyDetector = &emptyCustomAnomalyDetectorT{}

func customFactory(cfg Config) (AnomalyDetector, error) { return emptyCustomAnomalyDetector, nil }

func TestNew(t *testing.T) {

	devLogger, _ := zap.NewDevelopment()

	type args struct {
		cfg Config
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
				cfg: Config{},
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "custom",
			args: args{
				cfg: Config{customFactory: func(cfg Config) (AnomalyDetector, error) { return emptyCustomAnomalyDetector, nil }},
			},
			wantErr:      false,
			want:         emptyCustomAnomalyDetector,
			compareValue: true,
		},
		{
			name: "missing Param",
			args: args{
				cfg: Config{
					Logger:    devLogger,
					PodLister: nil,
					BreakerStrategyConfig: v1.BreakerStrategy{
						DiscreteValueOutOfList: &v1.DiscreteValueOutOfList{},
					},
				},
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "PROM: missing Service",
			args: args{
				cfg: Config{
					Logger:    devLogger,
					PodLister: nil,
					BreakerStrategyConfig: v1.BreakerStrategy{
						DiscreteValueOutOfList: &v1.DiscreteValueOutOfList{
							PromQL: "fake query",
						},
					},
				},
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "PROM: missing good and bad values",
			args: args{
				cfg: Config{
					Logger:    devLogger,
					PodLister: nil,
					BreakerStrategyConfig: v1.BreakerStrategy{
						DiscreteValueOutOfList: &v1.DiscreteValueOutOfList{
							PromQL:            "fake query",
							PrometheusService: "PrometheusService",
						},
					},
				},
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "PROM: good and bad values at the same time",
			args: args{
				cfg: Config{
					Logger:    devLogger,
					PodLister: nil,
					BreakerStrategyConfig: v1.BreakerStrategy{
						DiscreteValueOutOfList: &v1.DiscreteValueOutOfList{
							PromQL:            "fake query",
							PrometheusService: "PrometheusService",
							GoodValues:        []string{"1"},
							BadValues:         []string{"0"},
						},
					},
				},
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "PROM: good value only",
			args: args{
				cfg: Config{
					Logger:    devLogger,
					PodLister: nil,
					BreakerStrategyConfig: v1.BreakerStrategy{
						DiscreteValueOutOfList: &v1.DiscreteValueOutOfList{
							PromQL:            "fake query",
							PrometheusService: "PrometheusService",
							GoodValues:        []string{"1"},
						},
					},
				},
			},
			wantErr: false,
			want:    nil,
		},
		{
			name: "PROM: bad value only",
			args: args{
				cfg: Config{
					Logger:    devLogger,
					PodLister: nil,
					BreakerStrategyConfig: v1.BreakerStrategy{
						DiscreteValueOutOfList: &v1.DiscreteValueOutOfList{
							PromQL:            "fake query",
							PrometheusService: "PrometheusService",
							BadValues:         []string{"0"},
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

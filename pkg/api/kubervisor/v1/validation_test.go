package v1

import (
	"testing"
)

func Test_validateBreakerStrategy(t *testing.T) {

	tests := []struct {
		name    string
		s       BreakerStrategy
		wantErr bool
	}{
		{
			name:    "none",
			s:       BreakerStrategy{},
			wantErr: true,
		},
		{
			name: "DiscreteValueOutOfList empty",
			s: BreakerStrategy{
				DiscreteValueOutOfList: &DiscreteValueOutOfList{},
			},
			wantErr: true,
		},
		{
			name: "DiscreteValueOutOfList ok",
			s: BreakerStrategy{
				DiscreteValueOutOfList: &DiscreteValueOutOfList{
					PromQL:            "fake query",
					PrometheusService: "svc",
					GoodValues:        []string{"1"},
					Key:               "code",
					PodNameKey:        "podname",
				},
			},
			wantErr: false,
		},
		{
			name: "DiscreteValueOutOfList ok with badvalues",
			s: BreakerStrategy{
				DiscreteValueOutOfList: &DiscreteValueOutOfList{
					PromQL:            "fake query",
					PrometheusService: "svc",
					BadValues:         []string{"1"},
					Key:               "code",
					PodNameKey:        "podname",
				},
			},
			wantErr: false,
		},
		{
			name: "DiscreteValueOutOfList KO with badvalues and goovalues",
			s: BreakerStrategy{
				DiscreteValueOutOfList: &DiscreteValueOutOfList{
					PromQL:            "fake query",
					PrometheusService: "svc",
					BadValues:         []string{"1"},
					GoodValues:        []string{"1"},
					Key:               "code",
					PodNameKey:        "podname",
				},
			},
			wantErr: true,
		},
		{
			name: "DiscreteValueOutOfList PROM: missing Service",
			s: BreakerStrategy{
				DiscreteValueOutOfList: &DiscreteValueOutOfList{
					PromQL:     "fake query",
					GoodValues: []string{"1"},
					Key:        "code",
					PodNameKey: "podname",
				},
			},
			wantErr: true,
		},
		{
			name: "DiscreteValueOutOfList PROM: missing promQL",
			s: BreakerStrategy{
				DiscreteValueOutOfList: &DiscreteValueOutOfList{
					PrometheusService: "svc",
					GoodValues:        []string{"1"},
					Key:               "code",
					PodNameKey:        "podname",
				},
			},
			wantErr: true,
		},
		{
			name: "DiscreteValueOutOfList missing podnamekey",
			s: BreakerStrategy{
				DiscreteValueOutOfList: &DiscreteValueOutOfList{
					PrometheusService: "svc",
					PromQL:            "fake query",
					GoodValues:        []string{"1"},
					Key:               "code",
				},
			},
			wantErr: true,
		},
		{
			name: "DiscreteValueOutOfList missing metric key",
			s: BreakerStrategy{
				DiscreteValueOutOfList: &DiscreteValueOutOfList{
					PrometheusService: "svc",
					PromQL:            "fake query",
					GoodValues:        []string{"1"},
					PodNameKey:        "podname",
				},
			},
			wantErr: true,
		},
		{
			name: "DiscreteValueOutOfList default case",
			s: BreakerStrategy{
				DiscreteValueOutOfList: &DiscreteValueOutOfList{
					GoodValues: []string{"1"},
					Key:        "code",
					PodNameKey: "podname",
				},
			},
			wantErr: true,
		},
		{
			name: "ContinuousValueDeviation empty",
			s: BreakerStrategy{
				ContinuousValueDeviation: &ContinuousValueDeviation{},
			},
			wantErr: true,
		},
		{
			name: "ContinuousValueDeviation missing PodName",
			s: BreakerStrategy{
				ContinuousValueDeviation: &ContinuousValueDeviation{},
			},
			wantErr: true,
		},
		{
			name: "ContinuousValueDeviation missing PodName",
			s: BreakerStrategy{
				ContinuousValueDeviation: &ContinuousValueDeviation{
					MaxDeviationPercent: NewFloat64(50.0),
				},
			},
			wantErr: true,
		},
		{
			name: "ContinuousValueDeviation missing Deviation Quantity",
			s: BreakerStrategy{
				ContinuousValueDeviation: &ContinuousValueDeviation{
					PodNameKey: "pod",
				},
			},
			wantErr: true,
		},
		{
			name: "ContinuousValueDeviation missing PromQL and PromQlService",
			s: BreakerStrategy{
				ContinuousValueDeviation: &ContinuousValueDeviation{
					PodNameKey:          "pod",
					MaxDeviationPercent: NewFloat64(50.0),
				},
			},
			wantErr: true,
		},
		{
			name: "ContinuousValueDeviation missing PromQL",
			s: BreakerStrategy{
				ContinuousValueDeviation: &ContinuousValueDeviation{
					PodNameKey:          "pod",
					MaxDeviationPercent: NewFloat64(50.0),
					PrometheusService:   "service",
				},
			},
			wantErr: true,
		},
		{
			name: "ContinuousValueDeviation missing Prom service",
			s: BreakerStrategy{
				ContinuousValueDeviation: &ContinuousValueDeviation{
					PodNameKey:          "pod",
					MaxDeviationPercent: NewFloat64(50.0),
					PromQL:              "query",
				},
			},
			wantErr: true,
		},
		{
			name: "ContinuousValueDeviation complete",
			s: BreakerStrategy{
				ContinuousValueDeviation: &ContinuousValueDeviation{
					PodNameKey:          "pod",
					MaxDeviationPercent: NewFloat64(50.0),
					PromQL:              "query",
					PrometheusService:   "service",
				},
			},
			wantErr: false,
		},
		{
			name: "CustomService",
			s: BreakerStrategy{
				CustomService: "Custo",
			},
			wantErr: false,
		},
		{
			name: "to much",
			s: BreakerStrategy{
				CustomService: "Custo",
				DiscreteValueOutOfList: &DiscreteValueOutOfList{
					PromQL:            "fake query",
					PrometheusService: "svc",
					BadValues:         []string{"1"},
					Key:               "code",
					PodNameKey:        "podname",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateBreakerStrategy(tt.s); (err != nil) != tt.wantErr {
				t.Errorf("validateBreakerStrategy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

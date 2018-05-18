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
			s:       BreakerStrategy{Name: "aname"},
			wantErr: true,
		},
		{
			name: "DiscreteValueOutOfList empty",
			s: BreakerStrategy{
				Name: "avalidname",
				DiscreteValueOutOfList: &DiscreteValueOutOfList{},
			},
			wantErr: true,
		},
		{
			name: "DiscreteValueOutOfList ok",
			s: BreakerStrategy{
				Name: "avalidname",
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
				Name: "avalidname",
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
				Name: "avalidname",
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
				Name: "avalidname",
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
				Name: "avalidname",
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
				Name: "avalidname",
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
				Name: "avalidname",
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
				Name: "avalidname",
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
				Name: "avalidname",
				ContinuousValueDeviation: &ContinuousValueDeviation{},
			},
			wantErr: true,
		},
		{
			name: "ContinuousValueDeviation missing PodName",
			s: BreakerStrategy{
				Name: "avalidname",
				ContinuousValueDeviation: &ContinuousValueDeviation{},
			},
			wantErr: true,
		},
		{
			name: "ContinuousValueDeviation missing PodName",
			s: BreakerStrategy{
				Name: "avalidname",
				ContinuousValueDeviation: &ContinuousValueDeviation{
					MaxDeviationPercent: NewFloat64(50.0),
				},
			},
			wantErr: true,
		},
		{
			name: "ContinuousValueDeviation missing Deviation Quantity",
			s: BreakerStrategy{
				Name: "avalidname",
				ContinuousValueDeviation: &ContinuousValueDeviation{
					PodNameKey: "pod",
				},
			},
			wantErr: true,
		},
		{
			name: "ContinuousValueDeviation missing PromQL and PromQlService",
			s: BreakerStrategy{
				Name: "avalidname",
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
				Name: "avalidname",
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
				Name: "avalidname",
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
				Name: "avalidname",
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
				Name:          "avalidname",
				CustomService: "Custo",
			},
			wantErr: false,
		},
		{
			name: "Activator",
			s: BreakerStrategy{
				Name:          "avalidname",
				CustomService: "Custo",
				Activator:     &ActivatorStrategy{},
			},
			wantErr: false,
		},
		{
			name: "Small EvaluationPeriod",
			s: BreakerStrategy{
				Name:             "avalidname",
				EvaluationPeriod: NewFloat64(0.005),
				CustomService:    "Custo",
				Activator:        &ActivatorStrategy{},
			},
			wantErr: true,
		},
		{
			name: "Large EvaluationPeriod",
			s: BreakerStrategy{
				Name:             "avalidname",
				EvaluationPeriod: NewFloat64(5 * 24 * 3600),
				CustomService:    "Custo",
				Activator:        &ActivatorStrategy{},
			},
			wantErr: true,
		},
		{
			name: "to much",
			s: BreakerStrategy{
				Name:          "avalidname",
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

func TestValidateKubervisorServiceSpec(t *testing.T) {
	type args struct {
		s KubervisorServiceSpec
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				s: KubervisorServiceSpec{
					Breakers:         []BreakerStrategy{*DefaultBreakerStrategy(&BreakerStrategy{Name: "aname", CustomService: "Custo"})},
					DefaultActivator: *DefaultActivatorStrategy(&ActivatorStrategy{}),
					Service:          "servicefine",
				},
			},
			wantErr: false,
		},
		{
			name: "noservice",
			args: args{
				s: KubervisorServiceSpec{
					Breakers:         []BreakerStrategy{*DefaultBreakerStrategy(&BreakerStrategy{Name: "aname", CustomService: "Custo"})},
					DefaultActivator: *DefaultActivatorStrategy(&ActivatorStrategy{}),
				},
			},
			wantErr: true,
		},
		{
			name: "badservice",
			args: args{
				s: KubervisorServiceSpec{
					Breakers:         []BreakerStrategy{*DefaultBreakerStrategy(&BreakerStrategy{Name: "aname", CustomService: "Custo"})},
					DefaultActivator: *DefaultActivatorStrategy(&ActivatorStrategy{}),
					Service:          ";^%$#))(",
				},
			},
			wantErr: true,
		},
		{
			name: "nobreaker",
			args: args{
				s: KubervisorServiceSpec{
					DefaultActivator: *DefaultActivatorStrategy(&ActivatorStrategy{}),
					Service:          "servicefine",
				},
			},
			wantErr: true,
		},
		{
			name: "emptybreakers",
			args: args{
				s: KubervisorServiceSpec{
					Breakers:         []BreakerStrategy{},
					DefaultActivator: *DefaultActivatorStrategy(&ActivatorStrategy{}),
					Service:          "servicefine",
				},
			},
			wantErr: true,
		},
		{
			name: "badbreakers",
			args: args{
				s: KubervisorServiceSpec{
					Breakers:         []BreakerStrategy{BreakerStrategy{}},
					DefaultActivator: *DefaultActivatorStrategy(&ActivatorStrategy{}),
					Service:          "servicefine",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateKubervisorServiceSpec(tt.args.s); (err != nil) != tt.wantErr {
				t.Errorf("ValidateKubervisorServiceSpec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

package breaker

import (
	"testing"

	"github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
)

type emptyCustomBreakerT struct {
	SimilarConfig bool
}

func (e *emptyCustomBreakerT) Run(stop <-chan struct{}) {}
func (e *emptyCustomBreakerT) CompareConfig(specConfig *v1.BreakerStrategy) bool {
	return e.SimilarConfig
}
func (e *emptyCustomBreakerT) GetStatus() v1.BreakerStatus {
	return v1.BreakerStatus{}
}

var emptyCustomBreaker Breaker = &emptyCustomBreakerT{}

func customFactory(cfg FactoryConfig) (Breaker, error) { return emptyCustomBreaker, nil }
func TestNew(t *testing.T) {
	type args struct {
		cfg FactoryConfig
	}
	tests := []struct {
		name      string
		args      args
		checkFunc func(b Breaker) bool
		wantErr   bool
	}{
		{
			name: "ok",
			args: args{
				cfg: FactoryConfig{
					Config: Config{
						BreakerStrategyConfig: v1.BreakerStrategy{
							DiscreteValueOutOfList: &v1.DiscreteValueOutOfList{
								PromQL:            "query",
								PrometheusService: "Service",
								GoodValues:        []string{"ok"},
								Key:               "code",
								PodNameKey:        "podname",
							},
						},
					},
				},
			},
			wantErr: false,
			checkFunc: func(b Breaker) bool {
				_, ok := b.(*BreakerImpl)
				return ok
			},
		},
		{
			name: "bad breaker name",
			args: args{
				cfg: FactoryConfig{
					Config: Config{
						BreakerName: "b1!@#!",
					},
				},
			},
			wantErr: true,
		},

		{
			name: "error",
			args: args{
				cfg: FactoryConfig{},
			},
			wantErr:   true,
			checkFunc: nil,
		},
		{
			name: "custom",
			args: args{
				cfg: FactoryConfig{customFactory: func(cfg FactoryConfig) (Breaker, error) { return emptyCustomBreaker, nil }},
			},
			wantErr: false,
			checkFunc: func(b Breaker) bool {
				return b == emptyCustomBreaker
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if !tt.checkFunc(got) {
				t.Errorf("Bad type")
			}
		})
	}
}

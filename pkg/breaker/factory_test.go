package breaker

import (
	"testing"
)

type emptyCustomBreakerT struct {
}

func (e *emptyCustomBreakerT) Run(stop <-chan struct{}) error { return nil }

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
			name: "error",
			args: args{
				cfg: FactoryConfig{},
			},
			wantErr: false,
			checkFunc: func(b Breaker) bool {
				_, ok := b.(*BreakerImpl)
				return ok
			},
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
			if !tt.checkFunc(got) {
				t.Errorf("Bad type")
			}
		})
	}
}

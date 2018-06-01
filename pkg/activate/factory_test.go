package activate

import (
	"testing"

	"k8s.io/apimachinery/pkg/labels"

	api "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1alpha1"
)

type emptyCustomActivatorT struct {
	SimilarConfig bool
}

func (e *emptyCustomActivatorT) Run(stop <-chan struct{})      {}
func (e *emptyCustomActivatorT) GetStatus() api.PodCountStatus { return api.PodCountStatus{} }
func (e *emptyCustomActivatorT) CompareConfig(specStrategy *api.ActivatorStrategy, specSelector labels.Selector) bool {
	return e.SimilarConfig
}

var emptyCustomActivator Activator = &emptyCustomActivatorT{}

func customFactory(cfg FactoryConfig) (Activator, error) { return emptyCustomActivator, nil }
func TestNew(t *testing.T) {
	type args struct {
		cfg FactoryConfig
	}
	tests := []struct {
		name      string
		args      args
		checkFunc func(b Activator) bool
		wantErr   bool
	}{
		{
			name: "ok",
			args: args{
				cfg: FactoryConfig{
					Config: Config{
						ActivatorStrategyConfig: api.ActivatorStrategy{},
					},
				},
			},
			wantErr: false,
			checkFunc: func(b Activator) bool {
				_, ok := b.(*ActivatorImpl)
				return ok
			},
		},
		{
			name: "label",
			args: args{
				cfg: FactoryConfig{
					Config: Config{
						ActivatorStrategyConfig: api.ActivatorStrategy{},
						Selector:                labels.SelectorFromSet(map[string]string{"app": "foo"}),
					},
				},
			},
			wantErr: false,
			checkFunc: func(b Activator) bool {
				_, ok := b.(*ActivatorImpl)
				return ok
			},
		},
		{
			name: "custom",
			args: args{
				cfg: FactoryConfig{customFactory: func(cfg FactoryConfig) (Activator, error) { return emptyCustomActivator, nil }},
			},
			wantErr: false,
			checkFunc: func(b Activator) bool {
				return b == emptyCustomActivator
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

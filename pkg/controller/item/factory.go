package item

import (
	"fmt"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/labels"
	kv1 "k8s.io/client-go/listers/core/v1"

	"github.com/amadeusitgroup/podkubervisor/pkg/pod"

	activator "github.com/amadeusitgroup/podkubervisor/pkg/activate"
	apiv1 "github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
	"github.com/amadeusitgroup/podkubervisor/pkg/breaker"
)

// New return new BreakerConfigItem instance
func New(bc *apiv1.BreakerConfig, cfg *Config) (Interface, error) {
	if cfg.customFactory != nil {
		return cfg.customFactory(bc, cfg)
	}

	activateConfig := activator.FactoryConfig{
		Config: activator.Config{
			ActivatorStrategyConfig: bc.Spec.Activator,
			BreakerName:             fmt.Sprintf("%s/%s", bc.Namespace, bc.Name),
			PodControl:              cfg.PodControl,
			PodLister:               cfg.PodLister.Pods(bc.Namespace),
			Logger:                  cfg.Logger,
		},
	}
	activatorInterface, err := activator.New(activateConfig)
	if err != nil {
		return nil, err
	}

	breakerConfig := breaker.FactoryConfig{
		Config: breaker.Config{
			BreakerConfigName:     bc.Name,
			BreakerStrategyConfig: bc.Spec.Breaker,
			Selector:              cfg.Selector,
			PodControl:            cfg.PodControl,
			PodLister:             cfg.PodLister.Pods(bc.Namespace),
			Logger:                cfg.Logger,
		},
	}
	breakerInterface, err := breaker.New(breakerConfig)
	if err != nil {
		return nil, err
	}
	return &BreakerConfigItem{
		name:      fmt.Sprintf("%s/%s", bc.Namespace, bc.Name),
		activator: activatorInterface,
		breaker:   breakerInterface,
	}, nil

}

// Config Item factory configuration
type Config struct {
	Selector   labels.Selector
	PodLister  kv1.PodLister
	PodControl pod.ControlInterface
	Logger     *zap.Logger

	customFactory Factory
}

//Factory functor for Interface
type Factory func(bc *apiv1.BreakerConfig, cfg *Config) (Interface, error)

var _ Factory = New

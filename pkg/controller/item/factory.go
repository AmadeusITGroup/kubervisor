package item

import (
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/labels"
	kv1 "k8s.io/client-go/listers/core/v1"

	"github.com/amadeusitgroup/kubervisor/pkg/pod"

	activator "github.com/amadeusitgroup/kubervisor/pkg/activate"
	apiv1 "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1"
	"github.com/amadeusitgroup/kubervisor/pkg/breaker"
)

// New return new KubervisorServiceItem instance
func New(bc *apiv1.KubervisorService, cfg *Config) (Interface, error) {
	if cfg.customFactory != nil {
		return cfg.customFactory(bc, cfg)
	}

	activateDefaultConfig := activator.FactoryConfig{
		Config: activator.Config{
			ActivatorStrategyConfig: bc.Spec.DefaultActivator,
			Selector:                cfg.Selector,
			BreakerName:             bc.Name,
			PodControl:              cfg.PodControl,
			PodLister:               cfg.PodLister.Pods(bc.Namespace),
			Logger:                  cfg.Logger,
		},
	}
	activatorDefaultInterface, err := activator.New(activateDefaultConfig)
	if err != nil {
		return nil, err
	}

	baPairs := []breakerActivatorPair{}
	for _, bspec := range bc.Spec.Breakers {
		breakerConfig := breaker.FactoryConfig{
			Config: breaker.Config{
				KubervisorServiceName: bc.Name,
				BreakerStrategyConfig: bspec,
				StrategyName:          bspec.Name,
				Selector:              cfg.Selector,
				PodControl:            cfg.PodControl,
				PodLister:             cfg.PodLister.Pods(bc.Namespace),
				Logger:                cfg.Logger,
				BreakerName:           bc.Name,
			},
		}
		breakerInterface, err := breaker.New(breakerConfig)
		if err != nil {
			return nil, err
		}
		baPair := breakerActivatorPair{
			breaker: breakerInterface,
		}

		if bspec.Activator != nil {
			activateConfig := activator.FactoryConfig{
				Config: activator.Config{
					ActivatorStrategyConfig: *bspec.Activator,
					Selector:                cfg.Selector,
					BreakerName:             bc.Name,
					BreakerStrategyName:     bspec.Name,
					PodControl:              cfg.PodControl,
					PodLister:               cfg.PodLister.Pods(bc.Namespace),
					Logger:                  cfg.Logger,
				},
			}
			activatorInterface, err := activator.New(activateConfig)
			if err != nil {
				return nil, err
			}
			baPair.activator = activatorInterface
		}
		baPairs = append(baPairs, baPair)
	}

	return &KubervisorServiceItem{
		name:             bc.Name,
		namespace:        bc.Namespace,
		defaultActivator: activatorDefaultInterface,
		breakers:         baPairs,
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
type Factory func(bc *apiv1.KubervisorService, cfg *Config) (Interface, error)

var _ Factory = New

package item

import (
	"fmt"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/labels"
	kv1 "k8s.io/client-go/listers/core/v1"

	activator "github.com/amadeusitgroup/kubervisor/pkg/activate"
	apiv1 "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1alpha1"
	"github.com/amadeusitgroup/kubervisor/pkg/breaker"
	"github.com/amadeusitgroup/kubervisor/pkg/labeling"
	"github.com/amadeusitgroup/kubervisor/pkg/pod"
)

// New return new KubervisorServiceItem instance
func New(bc *apiv1.KubervisorService, cfg *Config) (Interface, error) {
	if cfg.customFactory != nil {
		return cfg.customFactory(bc, cfg)
	}

	namespacedPodLister := cfg.PodLister.Pods(bc.Namespace)
	augmentedSelector, errSelector := labeling.SelectorWithBreakerName(cfg.Selector, bc.Name)
	if errSelector != nil {
		return nil, fmt.Errorf("Can't build activator: %v", errSelector)
	}

	activateDefaultConfig := activator.FactoryConfig{
		Config: activator.Config{
			KubervisorName:          bc.Name,
			Selector:                augmentedSelector,
			ActivatorStrategyConfig: bc.Spec.DefaultActivator,
			PodControl:              cfg.PodControl,
			PodLister:               namespacedPodLister,
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
				KubervisorName:        bc.Name,
				StrategyName:          bspec.Name,
				Selector:              augmentedSelector,
				BreakerStrategyConfig: bspec,
				PodControl:            cfg.PodControl,
				PodLister:             namespacedPodLister,
				Logger:                cfg.Logger,
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
					KubervisorName:          bc.Name,
					BreakerStrategyName:     bspec.Name,
					Selector:                augmentedSelector,
					ActivatorStrategyConfig: *bspec.Activator,
					PodControl:              cfg.PodControl,
					PodLister:               namespacedPodLister,
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
		selector:         augmentedSelector,
		defaultActivator: activatorDefaultInterface,
		breakers:         baPairs,
		podLister:        namespacedPodLister,
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

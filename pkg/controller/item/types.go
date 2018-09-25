package item

import (
	"context"
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/labels"
	kv1 "k8s.io/client-go/listers/core/v1"
	kcache "k8s.io/client-go/tools/cache"

	activator "github.com/amadeusitgroup/kubervisor/pkg/activate"
	api "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1alpha1"
	"github.com/amadeusitgroup/kubervisor/pkg/breaker"
	"github.com/amadeusitgroup/kubervisor/pkg/labeling"
	"github.com/amadeusitgroup/kubervisor/pkg/pod"
)

// KubervisorServiceItemStore represent the KubervisorService Item store
type KubervisorServiceItemStore kcache.Store

// NewBreackerConfigItemStore returns new instance of a BreackerConfigItemStore
func NewBreackerConfigItemStore() KubervisorServiceItemStore {
	return KubervisorServiceItemStore(kcache.NewStore(KubervisorServiceItemKeyFunc))
}

// KubervisorServiceItemKeyFunc function used to return the key ok a KubervisorServiceItem instance
func KubervisorServiceItemKeyFunc(obj interface{}) (string, error) {
	bci, ok := obj.(Interface)
	if !ok {
		fmt.Printf("keyFunc obj:%v\n", obj)
		return "", fmt.Errorf("unable to return the key from obj: %v", obj)
	}

	return GetKey(bci.Namespace(), bci.Name()), nil
}

// GetKey return Item key from namespace name association
func GetKey(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

// Interface item interface
type Interface interface {
	Name() string
	Namespace() string
	Start(ctx context.Context)
	Stop() error
	CompareWithSpec(spec *api.KubervisorServiceSpec, selector labels.Selector) bool
	GetStatus() (api.PodCountStatus, error)
}

type breakerActivatorPair struct {
	activator activator.Activator
	breaker   breaker.Breaker
}

// CompareConfig compare the pair with the spec and return true if equal
func (p *breakerActivatorPair) CompareConfig(specBreaker *api.BreakerStrategy, specSelector labels.Selector) bool {
	//First compare the activator part
	if (p.activator != nil && specBreaker.Activator == nil) || (p.activator == nil && specBreaker.Activator != nil) {
		return false
	}
	if p.activator != nil && specBreaker.Activator != nil {
		if !p.activator.CompareConfig(specBreaker.Activator, specSelector) {
			return false
		}
	}
	//Second compare the breaker part
	return p.breaker.CompareConfig(specBreaker, specSelector)
}

//KubervisorServiceItem  Use to agreagate all sub process linked to a KubervisorService
type KubervisorServiceItem struct {
	// breaker config name
	name             string
	namespace        string
	defaultActivator activator.Activator
	breakers         []breakerActivatorPair

	podLister kv1.PodNamespaceLister
	selector  labels.Selector

	cancelFunc context.CancelFunc
	waitGroup  sync.WaitGroup
}

// Name returns the KubervisorServiceItem name
func (b *KubervisorServiceItem) Name() string {
	return b.name
}

// Namespace returns the KubervisorServiceItem name
func (b *KubervisorServiceItem) Namespace() string {
	return b.namespace
}

// Start used to star the Activator and Breaker
func (b *KubervisorServiceItem) Start(ctx context.Context) {
	var internalContext context.Context
	internalContext, b.cancelFunc = context.WithCancel(ctx)
	go b.runActivator(internalContext, b.defaultActivator)
	for _, baPair := range b.breakers {
		if baPair.activator != nil {
			go b.runActivator(internalContext, baPair.activator)
		}
		go b.runBreaker(internalContext, baPair.breaker)
	}
}

// Stop used to stop the Activator and Breaker
func (b *KubervisorServiceItem) Stop() error {
	if b.cancelFunc == nil {
		return nil
	}
	b.cancelFunc()
	b.waitGroup.Wait()
	return nil
}

func (b *KubervisorServiceItem) getBreakerByStrategyName(strategyName string) *breakerActivatorPair {
	for i := range b.breakers {
		if b.breakers[i].breaker.Name() == strategyName {
			return &b.breakers[i]
		}
	}
	return nil
}

// CompareWithSpec used to compare the current configuration with a new KubervisorServiceSpec (return true if there is a difference ... maybe the function need to be renamed)
func (b *KubervisorServiceItem) CompareWithSpec(spec *api.KubervisorServiceSpec, selector labels.Selector) bool {
	if !b.defaultActivator.CompareConfig(&spec.DefaultActivator, selector) {
		return true
	}
	if len(spec.Breakers) != len(b.breakers) {
		return true
	}
	for i, breakerStrategy := range spec.Breakers {
		breaker := b.getBreakerByStrategyName(breakerStrategy.Name)
		if breaker == nil {
			return true
		}
		if !breaker.CompareConfig(&spec.Breakers[i], selector) {
			return true
		}
	}
	return false
}

func (b *KubervisorServiceItem) runBreaker(ctx context.Context, breaker breaker.Breaker) {
	b.waitGroup.Add(1)
	defer b.waitGroup.Done()
	breaker.Run(ctx.Done())
}

func (b *KubervisorServiceItem) runActivator(ctx context.Context, activator activator.Activator) {
	b.waitGroup.Add(1)
	defer b.waitGroup.Done()
	activator.Run(ctx.Done())
}

//GetStatus return the status for the breaker
func (b *KubervisorServiceItem) GetStatus() (api.PodCountStatus, error) {
	status := api.PodCountStatus{}
	allPods, err := b.podLister.List(b.selector)
	if err != nil {
		return status, err
	}
	status.NbPodsManaged = uint32(len(allPods))
	for _, p := range allPods {
		if !pod.IsReady(p) {
			status.NbPodsManaged--
			continue
		}
		yesTraffic, pauseTraffic, err := labeling.IsPodTrafficLabelOkOrPause(p)
		switch {
		case err != nil:
			status.NbPodsUnknown++
		case pauseTraffic:
			status.NbPodsPaused++
		case !yesTraffic:
			status.NbPodsBreaked++
		}
	}
	return status, nil
}

package item

import (
	"context"
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/labels"
	kcache "k8s.io/client-go/tools/cache"

	activator "github.com/amadeusitgroup/podkubervisor/pkg/activate"
	"github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
	"github.com/amadeusitgroup/podkubervisor/pkg/breaker"
)

// BreakerConfigItemStore represent the BreakerConfig Item store
type BreakerConfigItemStore kcache.Store

// NewBreackerConfigItemStore returns new instance of a BreackerConfigItemStore
func NewBreackerConfigItemStore() BreakerConfigItemStore {
	return BreakerConfigItemStore(kcache.NewStore(keyFunc))
}

func keyFunc(obj interface{}) (string, error) {
	bci, ok := obj.(Interface)
	if !ok {
		return "", fmt.Errorf("unable to return the key from obj: %v", obj)
	}

	return bci.Name(), nil
}

// Interface item interface
type Interface interface {
	Name() string
	Start(ctx context.Context)
	Stop() error
	CompareWithSpec(spec *v1.BreakerConfigSpec, selector labels.Selector) bool
}

//BreakerConfigItem  Use to agreagate all sub process linked to a BreakerConfig
type BreakerConfigItem struct {
	// breaker config name
	name      string
	activator activator.Activator
	breaker   breaker.Breaker

	cancelFunc context.CancelFunc
	waitGroup  sync.WaitGroup
}

// Name returns the BreakerConfigItem name
func (b *BreakerConfigItem) Name() string {
	return b.name
}

// Start used to star the Activator and Breaker
func (b *BreakerConfigItem) Start(ctx context.Context) {
	var internalContext context.Context
	internalContext, b.cancelFunc = context.WithCancel(ctx)
	go b.runBreaker(internalContext)
	go b.runActivator(internalContext)
}

// Stop used to stop the Activator and Breaker
func (b *BreakerConfigItem) Stop() error {
	if b.cancelFunc == nil {
		return nil
	}
	b.cancelFunc()
	b.waitGroup.Wait()
	return nil
}

// CompareWithSpec used to compare the current configuration with a new BreakerConfigSpec
func (b *BreakerConfigItem) CompareWithSpec(spec *v1.BreakerConfigSpec, selector labels.Selector) bool {
	if !b.activator.CompareConfig(&spec.Activator, selector) {
		return false
	}
	if !b.breaker.CompareConfig(&spec.Breaker) {
		return false
	}

	return true
}

func (b *BreakerConfigItem) runBreaker(ctx context.Context) {
	b.waitGroup.Add(1)
	defer b.waitGroup.Done()
	b.breaker.Run(ctx.Done())
}

func (b *BreakerConfigItem) runActivator(ctx context.Context) {
	b.waitGroup.Add(1)
	defer b.waitGroup.Done()
	b.activator.Run(ctx.Done())
}

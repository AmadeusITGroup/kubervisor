package item

import (
	"context"
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/labels"
	kcache "k8s.io/client-go/tools/cache"

	activator "github.com/amadeusitgroup/kubervisor/pkg/activate"
	"github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1"
	"github.com/amadeusitgroup/kubervisor/pkg/breaker"
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

	return fmt.Sprintf("%s/%s", bci.Namespace(), bci.Name()), nil
}

// Interface item interface
type Interface interface {
	Name() string
	Namespace() string
	Start(ctx context.Context)
	Stop() error
	CompareWithSpec(spec *v1.KubervisorServiceSpec, selector labels.Selector) bool
	GetStatus() v1.BreakerStatus
}

//KubervisorServiceItem  Use to agreagate all sub process linked to a KubervisorService
type KubervisorServiceItem struct {
	// breaker config name
	name      string
	namespace string
	activator activator.Activator
	breaker   breaker.Breaker

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
	go b.runBreaker(internalContext)
	go b.runActivator(internalContext)
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

// CompareWithSpec used to compare the current configuration with a new KubervisorServiceSpec
func (b *KubervisorServiceItem) CompareWithSpec(spec *v1.KubervisorServiceSpec, selector labels.Selector) bool {
	if !b.activator.CompareConfig(&spec.Activator, selector) {
		return true
	}
	if !b.breaker.CompareConfig(&spec.Breaker) {
		return true
	}

	return false
}

func (b *KubervisorServiceItem) runBreaker(ctx context.Context) {
	b.waitGroup.Add(1)
	defer b.waitGroup.Done()
	b.breaker.Run(ctx.Done())
}

func (b *KubervisorServiceItem) runActivator(ctx context.Context) {
	b.waitGroup.Add(1)
	defer b.waitGroup.Done()
	b.activator.Run(ctx.Done())
}

//GetStatus get the current status for the breaker (stats on pods)
func (b *KubervisorServiceItem) GetStatus() v1.BreakerStatus {
	return b.breaker.GetStatus()
}

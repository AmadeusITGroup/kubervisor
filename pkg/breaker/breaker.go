package breaker

import (
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/labels"
	kv1 "k8s.io/client-go/listers/core/v1"

	"github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
)

//Breaker engine that check anomaly and relabel pods
type Breaker interface {
	Run(stop <-chan struct{}) error
}

type Config struct {
	BreakerStrategyConfig v1.BreakerStrategy
	Selector              labels.Selector
	PodLister             kv1.PodLister
	Logger                *zap.Logger
}

type BreakerImpl struct {
}

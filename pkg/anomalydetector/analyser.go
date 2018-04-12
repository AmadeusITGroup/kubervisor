package anomalydetector

import (
	"go.uber.org/zap"
	kapiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	kv1 "k8s.io/client-go/listers/core/v1"

	"github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
)

//AnomalyDetector returns the list of pods that do not behave correctly according to the configuration
type AnomalyDetector interface {
	GetPodsOutOfBounds() ([]*kapiv1.Pod, error)
}

//Config parameters required for the creation of an AnomalyDetector
type Config struct {
	BreakerStrategyConfig v1.BreakerStrategy
	Selector              labels.Selector
	PodLister             kv1.PodNamespaceLister
	Logger                *zap.Logger
}

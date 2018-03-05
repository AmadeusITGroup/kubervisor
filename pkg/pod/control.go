package pod

import (
	"strconv"
	"time"

	kapiv1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"

	"github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
	"github.com/amadeusitgroup/podkubervisor/pkg/labeling"
)

//ControlInterface interface to act on pods
type ControlInterface interface {
	UpdateBreakerAnnotationAndLabel(p *kapiv1.Pod) (*kapiv1.Pod, error)
}

var _ ControlInterface = &Control{}

//Control implements pod controlInterface
type Control struct {
	kubeClient    clientset.Interface
	breakerConfig v1.BreakerConfig
}

//UpdateBreakerAnnotationAndLabel implements pod control
func (c *Control) UpdateBreakerAnnotationAndLabel(inputPod *kapiv1.Pod) (*kapiv1.Pod, error) {
	p := *inputPod //Copy to avoid modifying object inside the cache

	if p.Annotations == nil {
		p.Annotations = map[string]string{}
	}
	if p.Labels == nil {
		p.Labels = map[string]string{}
	}

	p.Labels[labeling.LabelBreakerNameKey] = c.breakerConfig.Name
	p.Labels[labeling.LabelTrafficKey] = string(labeling.LabelTrafficNo)

	retryCount, _ := labeling.GetRetryCount(&p)
	retryCount++

	p.Annotations[labeling.AnnotationBreakAtKey] = time.Now().Format(time.RFC3339)
	p.Annotations[labeling.AnnotationRetryCountKey] = strconv.Itoa(retryCount)

	return c.kubeClient.Core().Pods(p.Namespace).Update(&p)
}

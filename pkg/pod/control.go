package pod

import (
	"strconv"
	"time"

	kapiv1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"

	"github.com/amadeusitgroup/podkubervisor/pkg/labeling"
)

//ControlInterface interface to act on pods
type ControlInterface interface {
	InitBreakerAnnotationAndLabel(breakConfigName string, inputPod *kapiv1.Pod) (*kapiv1.Pod, error)
	UpdateBreakerAnnotationAndLabel(breakConfigName string, p *kapiv1.Pod) (*kapiv1.Pod, error)
	UpdateActivationLabelsAndAnnotations(p *kapiv1.Pod) (*kapiv1.Pod, error)
	UpdatePauseLabelsAndAnnotations(p *kapiv1.Pod) (*kapiv1.Pod, error)
	RemoveBreakerAnnotationAndLabel(p *kapiv1.Pod) (*kapiv1.Pod, error)
	KillPod(p *kapiv1.Pod) error
}

var _ ControlInterface = &Control{}

// NewPodControl returns new PodControl instance
func NewPodControl(client clientset.Interface) *Control {
	return &Control{kubeClient: client}
}

//Control implements pod controlInterface
type Control struct {
	kubeClient clientset.Interface
}

func copyAndDefault(inputPod *kapiv1.Pod) *kapiv1.Pod {
	p := inputPod.DeepCopy()
	if p.Annotations == nil {
		p.Annotations = map[string]string{}
	}
	if p.Labels == nil {
		p.Labels = map[string]string{}
	}
	return p
}

//InitBreakerAnnotationAndLabel implements pod control
func (c *Control) InitBreakerAnnotationAndLabel(breakConfigName string, inputPod *kapiv1.Pod) (*kapiv1.Pod, error) {
	//Copy to avoid modifying object inside the cache
	p := copyAndDefault(inputPod)

	p.Labels[labeling.LabelBreakerNameKey] = breakConfigName
	labeling.SetTrafficLabel(p, labeling.LabelTrafficYes)
	return c.kubeClient.Core().Pods(p.Namespace).Update(p)
}

//UpdateBreakerAnnotationAndLabel implements pod control
func (c *Control) UpdateBreakerAnnotationAndLabel(breakConfigName string, inputPod *kapiv1.Pod) (*kapiv1.Pod, error) {
	//Copy to avoid modifying object inside the cache
	p := copyAndDefault(inputPod)

	p.Labels[labeling.LabelBreakerNameKey] = breakConfigName
	p.Labels[labeling.LabelTrafficKey] = string(labeling.LabelTrafficNo)

	retryCount, _ := labeling.GetRetryCount(p)
	retryCount++

	p.Annotations[labeling.AnnotationBreakAtKey] = time.Now().Format(time.RFC3339)
	p.Annotations[labeling.AnnotationRetryCountKey] = strconv.Itoa(retryCount)

	return c.kubeClient.Core().Pods(p.Namespace).Update(p)
}

//UpdateActivationLabelsAndAnnotations classic pod reactivation into traffic
func (c *Control) UpdateActivationLabelsAndAnnotations(inputPod *kapiv1.Pod) (*kapiv1.Pod, error) {
	//Copy to avoid modifying object inside the cache
	p := copyAndDefault(inputPod)

	p.Labels[labeling.LabelTrafficKey] = string(labeling.LabelTrafficYes)

	return c.kubeClient.Core().Pods(p.Namespace).Update(p)
}

//UpdatePauseLabelsAndAnnotations called to put pod on pause when count exceeded
func (c *Control) UpdatePauseLabelsAndAnnotations(inputPod *kapiv1.Pod) (*kapiv1.Pod, error) {
	//Copy to avoid modifying object inside the cache
	p := copyAndDefault(inputPod)

	p.Labels[labeling.LabelTrafficKey] = string(labeling.LabelTrafficPause)

	return c.kubeClient.Core().Pods(p.Namespace).Update(p)
}

// RemoveBreakerAnnotationAndLabel called to remove all labels and annotations added previously.
func (c *Control) RemoveBreakerAnnotationAndLabel(inputPod *kapiv1.Pod) (*kapiv1.Pod, error) {
	p := copyAndDefault(inputPod)

	delete(p.Labels, labeling.LabelBreakerNameKey)

	delete(p.Annotations, labeling.AnnotationBreakAtKey)
	delete(p.Annotations, labeling.AnnotationRetryCountKey)

	return c.kubeClient.Core().Pods(p.Namespace).Update(p)
}

//KillPod deelte the pod. Called when the number of retry have been exceeded on a retyrAndKill strategy
func (c *Control) KillPod(inputPod *kapiv1.Pod) error {
	return c.kubeClient.Core().Pods(inputPod.Namespace).Delete(inputPod.Name, nil)
}

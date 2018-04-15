package pod

import (
	"strconv"
	"time"

	kapiv1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"

	"github.com/amadeusitgroup/podkubervisor/pkg/labeling"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	prometheus.MustRegister(kubervisorBreakCounters)
}

var (
	kubervisorBreakCounters = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubervisor_breaker_count",
			Help: "Count Pod under kubervisor management",
		},
		[]string{"breaker", "namespace", "pod", "type"}, // type={managed,breaked,paused,unknown}
	)
)

//ControlInterface interface to act on pods
type ControlInterface interface {
	InitBreakerAnnotationAndLabel(breakConfigName string, inputPod *kapiv1.Pod) (*kapiv1.Pod, error)
	UpdateBreakerAnnotationAndLabel(breakConfigName string, p *kapiv1.Pod) (*kapiv1.Pod, error)
	UpdateActivationLabelsAndAnnotations(breakConfigName string, p *kapiv1.Pod) (*kapiv1.Pod, error)
	UpdatePauseLabelsAndAnnotations(breakConfigName string, p *kapiv1.Pod) (*kapiv1.Pod, error)
	RemoveBreakerAnnotationAndLabel(p *kapiv1.Pod) (*kapiv1.Pod, error)
	KillPod(breakConfigName string, p *kapiv1.Pod) error
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
func (c *Control) UpdateBreakerAnnotationAndLabel(breakConfigName string, inputPod *kapiv1.Pod) (returnPod *kapiv1.Pod, err error) {
	//Copy to avoid modifying object inside the cache
	p := copyAndDefault(inputPod)

	p.Labels[labeling.LabelBreakerNameKey] = breakConfigName
	p.Labels[labeling.LabelTrafficKey] = string(labeling.LabelTrafficNo)

	retryCount, _ := labeling.GetRetryCount(p)
	retryCount++

	p.Annotations[labeling.AnnotationBreakAtKey] = time.Now().Format(time.RFC3339)
	p.Annotations[labeling.AnnotationRetryCountKey] = strconv.Itoa(retryCount)

	returnPod, err = c.kubeClient.Core().Pods(p.Namespace).Update(p)
	if err != nil {
		kubervisorBreakCounters.WithLabelValues(breakConfigName, p.Namespace, p.Name, "break").Inc()
	}
	return
}

//UpdateActivationLabelsAndAnnotations classic pod reactivation into traffic
func (c *Control) UpdateActivationLabelsAndAnnotations(breakerConfigName string, inputPod *kapiv1.Pod) (returnPod *kapiv1.Pod, err error) {
	//Copy to avoid modifying object inside the cache
	p := copyAndDefault(inputPod)

	p.Labels[labeling.LabelTrafficKey] = string(labeling.LabelTrafficYes)

	returnPod, err = c.kubeClient.Core().Pods(p.Namespace).Update(p)
	if err != nil {
		kubervisorBreakCounters.WithLabelValues(breakerConfigName, p.Namespace, p.Name, "activate").Inc()
	}
	return
}

//UpdatePauseLabelsAndAnnotations called to put pod on pause when count exceeded
func (c *Control) UpdatePauseLabelsAndAnnotations(breakerConfigName string, inputPod *kapiv1.Pod) (returnPod *kapiv1.Pod, err error) {
	//Copy to avoid modifying object inside the cache
	p := copyAndDefault(inputPod)

	p.Labels[labeling.LabelTrafficKey] = string(labeling.LabelTrafficPause)

	returnPod, err = c.kubeClient.Core().Pods(p.Namespace).Update(p)
	if err != nil {
		kubervisorBreakCounters.WithLabelValues(breakerConfigName, p.Namespace, p.Name, "pause").Inc()
	}
	return
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
func (c *Control) KillPod(breakerConfigName string, inputPod *kapiv1.Pod) error {
	err := c.kubeClient.Core().Pods(inputPod.Namespace).Delete(inputPod.Name, nil)
	if err != nil {
		kubervisorBreakCounters.WithLabelValues(breakerConfigName, inputPod.Namespace, inputPod.Name, "kill").Inc()
	}
	return err
}

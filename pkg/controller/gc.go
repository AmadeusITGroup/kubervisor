package controller

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"k8s.io/apimachinery/pkg/labels"
	corev1listers "k8s.io/client-go/listers/core/v1"

	blisters "github.com/amadeusitgroup/kubervisor/pkg/client/listers/kubervisor/v1"
	"github.com/amadeusitgroup/kubervisor/pkg/labeling"
	"github.com/amadeusitgroup/kubervisor/pkg/pod"
)

//garbageCollector runs against pods and kubervisorServices. If pods refers a KubervisorService that does not exist anymore then the labels on that pod are cleaned.
type garbageCollector struct {
	logger            *zap.Logger
	podLister         corev1listers.PodLister
	podControl        pod.ControlInterface
	breakerLister     blisters.KubervisorServiceLister
	period            time.Duration
	missCountBeforeGC int
	counters          map[string]int
}

func newGarbageCollector(period time.Duration, podControl pod.ControlInterface, podLister corev1listers.PodLister, breakerLister blisters.KubervisorServiceLister, missCountBeforeGC int, logger *zap.Logger) (*garbageCollector, error) {

	if logger == nil || podLister == nil || podControl == nil || breakerLister == nil || period.Seconds() == 0.0 || missCountBeforeGC < 1 {
		return nil, fmt.Errorf("Bad GC parameter(s)")
	}

	return &garbageCollector{
		period:            period,
		logger:            logger,
		podControl:        podControl,
		podLister:         podLister,
		breakerLister:     breakerLister,
		missCountBeforeGC: missCountBeforeGC,
		counters:          map[string]int{},
	}, nil
}

func (gc *garbageCollector) run(stop <-chan struct{}) {
	ticker := time.NewTicker(gc.period)
	for {
		select {
		case <-ticker.C:
			gc.updateCounters()
			gc.cleanPods()

		case <-stop:
			return
		}
	}
}

func (gc *garbageCollector) updateCounters() {
	//Collect and index KubervisorService name
	kubervisorServices, err := gc.breakerLister.List(labels.Everything())
	if err != nil {
		gc.logger.Sugar().Errorf("GC Can't list kubervisor services: %v", err)
		return
	}
	kubervisorServiceName := map[string]struct{}{}
	for _, ksvc := range kubervisorServices {
		name := ksvc.Namespace + "/" + ksvc.Name
		kubervisorServiceName[name] = struct{}{}
	}

	//Check if for each pod with Kubervisor labels, the KubervisorService exist, if not update counter
	pods, err := gc.podLister.List(labels.Everything())
	if err != nil {
		gc.logger.Sugar().Errorf("GC Can't list pods: %v", err)
		return
	}
	for _, pod := range pods {
		if pod.Labels == nil {
			continue
		}
		ksvcName, ok := pod.Labels[labeling.LabelBreakerNameKey]
		if !ok {
			continue
		}
		if _, found := kubervisorServiceName[pod.Namespace+"/"+ksvcName]; !found {
			key := pod.Namespace + "/" + pod.Name
			count := gc.counters[key]
			count++
			gc.counters[key] = count
		}
	}
}

func (gc *garbageCollector) cleanPods() {
	toClean := []string{}
	for key, count := range gc.counters {
		if count >= gc.missCountBeforeGC {
			token := strings.Split(key, "/")
			if len(token) != 2 {
				gc.logger.Sugar().Errorf("GC Bad key structure %s. Key is ignored.", key)
				continue
			}
			p, err := gc.podLister.Pods(token[0]).Get(token[1])
			if err != nil || p == nil {
				gc.logger.Sugar().Errorf("GC Can't get pod %s. Error: %v", key, err)
				continue

			}
			if _, err := gc.podControl.RemoveBreakerAnnotationAndLabel(p); err != nil {
				gc.logger.Sugar().Errorf("GC Can't do clean up on pod %s/%s. Maybe at next iteration (%d). Error: %v", p.Namespace, p.Name, count, err)
				continue
			}
			toClean = append(toClean, key)
		}
	}
	for _, key := range toClean {
		delete(gc.counters, key)
	}
}

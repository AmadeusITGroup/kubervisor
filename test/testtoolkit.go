package utils_test

// Functions in this package are for unittest usage only
// FOR TEST PURPOSE ONLY

//folder name "test" and package "utils_test" are different on purpose: avoid automatic import by go_import and force import alias.

import (
	"sync"
	"testing"
	"time"

	kapiv1 "k8s.io/api/core/v1"
	kv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/amadeusitgroup/podkubervisor/pkg/labeling"
)

//ValidateTestSequence is a test helper to validate the sequence of steps
// FOR TEST PURPOSE ONLY
func ValidateTestSequence(wg *sync.WaitGroup, t *testing.T, duration time.Duration, sequenceTitle string, closingChannels []chan struct{}) {
	wg.Add(1)
	go func() {
		sequenceCompleted := make(chan struct{})
		go func() {
			defer wg.Done()
			timeout := time.After(duration)
			select {
			case <-timeout:
				t.Errorf("The sequence %s did not complete in %f s", sequenceTitle, duration.Seconds())
			case <-sequenceCompleted:
				return
			}
		}()

		for _, c := range closingChannels {
			<-c
		}
		close(sequenceCompleted)
	}()
}

type testStep struct {
	c        chan struct{}
	doneOnce bool
}

//TestStepSequence sequence of test steps
type TestStepSequence struct {
	name     string
	t        *testing.T
	steps    []testStep
	duration time.Duration
}

//PassOnlyOnce to be called when a step in the sequence is considered as passed and can't be passed a second time else (t.Fatal)
func (es *TestStepSequence) PassOnlyOnce(step int) {
	defer func() {
		if r := recover(); r != nil {
			es.t.Fatalf("Recovered in Passing step: %s", r)
			return
		}
	}()

	if step > len(es.steps) {
		es.t.Fatalf("Step out of bound")
		return
	}
	close(es.steps[step].c)
	es.steps[step].c = nil
	es.steps[step].doneOnce = true
}

//PassOnlyOnce to be called when a step in the sequence is considered as passed qnd can't be passed a second time
func (es *TestStepSequence) PassAtLeastOnce(step int) {
	defer func() {
		if r := recover(); r != nil {
			es.t.Fatalf("Recovered in Passing step: %s", r)
			return
		}
	}()

	if step > len(es.steps) {
		es.t.Fatalf("Step out of bound")
		return
	}

	if es.steps[step].doneOnce {
		return
	}
	close(es.steps[step].c)
	es.steps[step].c = nil
	es.steps[step].doneOnce = true
}

//ValidateTestSequence validate that the sequence is completed in order in the given time
func (es *TestStepSequence) ValidateTestSequence(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		sequenceCompleted := make(chan struct{})
		go func() {
			defer wg.Done()
			timeout := time.After(es.duration)
			select {
			case <-timeout:
				es.t.Errorf("The sequence %s did not complete in %f s", es.name, es.duration.Seconds())
			case <-sequenceCompleted:
				return
			}
		}()

		for _, step := range es.steps {
			<-step.c
		}
		close(sequenceCompleted)
	}()
}

//ValidateTestSequenceNoOrder validate that the sequence is completed in order in the given time
func (es *TestStepSequence) ValidateTestSequenceNoOrder(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		sequenceCompleted := make(chan struct{})
		go func() {
			defer wg.Done()
			timeout := time.After(es.duration)
			select {
			case <-timeout:
				es.t.Errorf("The sequence %s did not complete in %f s", es.name, es.duration.Seconds())
			case <-sequenceCompleted:
				return
			}
		}()
		var wgIn sync.WaitGroup
		for _, step := range es.steps {
			wgIn.Add(1)
			go func(ts testStep) {
				defer wgIn.Done()
				<-ts.c
			}(step)
		}
		wgIn.Wait()
		close(sequenceCompleted)
	}()
}

//NewTestSequence Create a test sequence
func NewTestSequence(t *testing.T, registrationName string, count int, duration time.Duration) *TestStepSequence {
	s := &TestStepSequence{
		name:     registrationName,
		steps:    make([]testStep, count),
		t:        t,
		duration: duration,
	}
	for i := range s.steps {
		s.steps[i].c = make(chan struct{})
	}

	if _, ok := MapOfSequences[s.name]; ok {
		t.Fatalf("Multiple definition of sequence %s", s.name)
		return nil
	}
	MapOfSequences[s.name] = s
	return s
}

//GetTestSequence retrieve test sequence
//Should be called
func GetTestSequence(t *testing.T, registrationName string) *TestStepSequence {
	//don't use t.Name because the name change depending if the testcase is running or not.
	s, ok := MapOfSequences[registrationName]
	if !ok {
		t.Fatalf("Undefined test sequence %s", registrationName)
	}
	return s
}

var MapOfSequences = map[string]*TestStepSequence{}

//NewTestPodLister create a new PodLister.
// FOR TEST PURPOSE ONLY
func NewTestPodLister(pods []*kapiv1.Pod) kv1.PodLister {
	index := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	for _, p := range pods {
		index.Add(p)
	}
	return kv1.NewPodLister(index)
}

//PodGen generate a pod with some label and status
// FOR TEST PURPOSE ONLY
func PodGen(name string, labels map[string]string, running, ready bool, trafficLabel labeling.LabelTraffic) *kapiv1.Pod {
	p := kapiv1.Pod{}
	p.Name = name
	if trafficLabel != "" {
		if labels == nil {
			labels = map[string]string{}
		}
		p.SetLabels(labels)
		labeling.SetTraficLabel(&p, trafficLabel)
	}
	if running {
		p.Status = kapiv1.PodStatus{Phase: kapiv1.PodRunning}
		if ready {
			p.Status.Conditions = []kapiv1.PodCondition{{Type: kapiv1.PodReady, Status: kapiv1.ConditionTrue}}
		} else {
			p.Status.Conditions = []kapiv1.PodCondition{{Type: kapiv1.PodReady, Status: kapiv1.ConditionFalse}}
		}
	} else {
		p.Status = kapiv1.PodStatus{Phase: kapiv1.PodUnknown}
	}
	return &p
}

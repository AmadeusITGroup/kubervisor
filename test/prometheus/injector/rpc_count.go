package main

import (
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type injector1 struct {
	name            string
	rpcCountAllOK   *prometheus.CounterVec
	rpcCountAllKO   *prometheus.CounterVec
	rpcCountBKO1min *prometheus.CounterVec
	step            int
}

func init() {
	inj := &injector1{name: "injector1"}

	//registration in prometheus client
	inj.rpcCountAllOK = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rpc_count_all_ok",
			Help: "Count RPC",
		},
		[]string{"podname", "scenario", "returncode"},
	)
	prometheus.MustRegister(inj.rpcCountAllOK)

	inj.rpcCountAllKO = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rpc_count_all_ko",
			Help: "Count RPC",
		},
		[]string{"podname", "scenario", "returncode"},
	)
	prometheus.MustRegister(inj.rpcCountAllKO)

	inj.rpcCountBKO1min = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rpc_count_B_KO_1min",
			Help: "Count RPC",
		},
		[]string{"podname", "scenario", "returncode"},
	)
	prometheus.MustRegister(inj.rpcCountBKO1min)

	//registration in injector engine

	if _, ok := injectorRegistry[inj.name]; ok {
		log.Fatalf("Name conflicts on injector:%s\n", inj.name)
	}
	injectorRegistry[inj.name] = inj
}

func (i *injector1) Run(stop <-chan struct{}) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			log.Println("Stopping injector " + i.name)
			return
		case <-ticker.C:
			i.scenarioAllOK()
			i.scenarioAllKO()
			i.scenarioBKO1Min()
			i.step++
		}
	}
}

func (i *injector1) scenarioAllOK() {
	i.rpcCountAllOK.WithLabelValues(i.name+"_A", "allOK", "200").Inc()
	i.rpcCountAllOK.WithLabelValues(i.name+"_B", "allOK", "200").Inc()
}

func (i *injector1) scenarioAllKO() {
	i.rpcCountAllKO.WithLabelValues(i.name+"_A", "allKO", "500").Inc()
	i.rpcCountAllKO.WithLabelValues(i.name+"_B", "allKO", "500").Inc()
}

func (i *injector1) scenarioBKO1Min() {
	switch {
	case i.step < 60:
		i.rpcCountBKO1min.WithLabelValues(i.name+"_A", "B_KO_1Min", "200").Inc()
		i.rpcCountBKO1min.WithLabelValues(i.name+"_B", "B_KO_1Min", "200").Inc()
	case i.step >= 60 && i.step < 120:
		i.rpcCountBKO1min.WithLabelValues(i.name+"_A", "B_KO_1Min", "200").Inc()
		i.rpcCountBKO1min.WithLabelValues(i.name+"_B", "B_KO_1Min", "500").Inc()
	case i.step >= 120:
		i.rpcCountBKO1min.WithLabelValues(i.name+"_A", "B_KO_1Min", "200").Inc()
		i.rpcCountBKO1min.WithLabelValues(i.name+"_B", "B_KO_1Min", "200").Inc()
	default:
		log.Println("Hole in metric report for injector1")
	}
}

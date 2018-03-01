package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	records = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "test1",
		Help: "Test for basic scenario.",
	})
)

var injectorRegistry = map[string]injector{}

type injector interface {
	Run(stop <-chan struct{})
}

var (
	addr = flag.String("listen-address", ":9102", "The address to listen on for HTTP requests.")
)

func main() {
	flag.Parse()
	stop := make(chan struct{})
	go runMetricsServer(stop)

	for name, inj := range injectorRegistry {
		log.Printf("Running injector %s\n", name)
		go inj.Run(stop)
	}

	<-time.After(3 * time.Minute)
	log.Printf("Sending stop signal and 10s graceful stop period\n")
	close(stop)
	time.Sleep(10 * time.Second)
	log.Printf("End of main\n")
}

func runMetricsServer(stop <-chan struct{}) {
	http.Handle("/metrics", promhttp.Handler())
	server := &http.Server{
		Addr:    *addr,
		Handler: http.DefaultServeMux,
	}

	go func() {
		log.Printf("Starting http server for metrics\n")
		log.Fatal(server.ListenAndServe())
	}()

	<-stop
	ctx, cf := context.WithTimeout(context.Background(), 5*time.Second)
	if cf != nil {
		cf()
	}
	server.Shutdown(ctx)
}

package main

import (
	"context"
	goflag "flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/heptiolabs/healthcheck"
	"github.com/spf13/pflag"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/amadeusitgroup/kubervisor/examples/demo/pkg/pricer"
	"github.com/amadeusitgroup/kubervisor/examples/demo/pkg/utils"
)

func main() {
	if err := run(); err != nil {
		log.Printf("exit with error: %v", err)
		os.Exit(1)
	}
	log.Println("shutting down")
	os.Exit(0)
}

func run() error {
	utils.BuildInfos()
	health := healthcheck.NewHandler()
	promRegistry := prometheus.NewRegistry()

	cfg := pricer.NewConfig(promRegistry)
	cfg.AddFlags(pflag.CommandLine)
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	pflag.Parse()
	goflag.CommandLine.Parse([]string{})

	pricerSvc := pricer.NewService(cfg)

	r := mux.NewRouter()
	r.HandleFunc("/api/v1/search", pricerSvc.SearchHandler).Methods("GET")
	r.HandleFunc("/setconfig", pricerSvc.SetConfig).Methods("POST")
	r.HandleFunc("/live", http.HandlerFunc(health.LiveEndpoint))
	r.HandleFunc("/ready", http.HandlerFunc(health.ReadyEndpoint))
	r.Handle("/metrics", promhttp.HandlerFor(promRegistry, promhttp.HandlerOpts{}))
	http.Handle("/", r)

	srv := &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%s", cfg.Port),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r, // Pass our instance of gorilla/mux in.
	}

	go func() {
		fmt.Println("ListenAndServe:", srv.Addr)
		if err := srv.ListenAndServe(); err != nil {
			log.Println("Http server error: ", err)
		}
	}()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c
	fmt.Println("Received signal...")
	return srv.Shutdown(context.Background())
}

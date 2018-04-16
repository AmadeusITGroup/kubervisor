package controller

import (
	"context"
	"fmt"
	"net/http"

	"github.com/heptiolabs/healthcheck"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (ctrl *Controller) runHTTPServer(stop <-chan struct{}) error {
	sugar := ctrl.Logger.Sugar()
	go func() {
		sugar.Infof("Listening on http://%s", ctrl.httpServer.Addr)

		if err := ctrl.httpServer.ListenAndServe(); err != nil {
			sugar.Error("Http server error: ", err)
		}
	}()

	<-stop
	sugar.Info("Shutting down the http server...")
	return ctrl.httpServer.Shutdown(context.Background())
}

func (ctrl *Controller) configureHTTPServer() {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/", ctrl.configureHealth())
	ctrl.httpServer.Handler = mux
}

func (ctrl *Controller) configureHealth() http.Handler {
	ctrl.health = healthcheck.NewHandler()
	ctrl.health.AddReadinessCheck("KubervisorService_cache_sync", func() error {
		if ctrl.BreakerSynced() {
			return nil
		}
		return fmt.Errorf("KubervisorService cache not sync")
	})
	ctrl.health.AddReadinessCheck("Pod_cache_sync", func() error {
		if ctrl.PodSynced() {
			return nil
		}
		return fmt.Errorf("Pod cache not sync")
	})
	ctrl.health.AddReadinessCheck("Service_cache_sync", func() error {
		if ctrl.ServiceSynced() {
			return nil
		}
		return fmt.Errorf("Service cache not sync")
	})

	return ctrl.health
}

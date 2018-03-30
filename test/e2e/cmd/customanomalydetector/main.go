package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/amadeusitgroup/podkubervisor/pkg/utils"
)

var closeChan chan struct{}
var response []byte

func main() {
	utils.BuildInfos()
	initGlobals()
	closeChan = make(chan struct{})

	go prepareResponse(time.Second)
	go runServer(serverAddr, getMux())

	<-closeChan
}

func prepareResponse(period time.Duration) {
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	options := meta_v1.ListOptions{
		LabelSelector: selector,
	}

	//Encoder
	gv := schema.GroupVersion{Group: "", Version: "v1"}
	codecFactory := serializer.NewCodecFactory(scheme.Scheme)
	mediaType := "application/json"
	info, ok := runtime.SerializerInfoForMediaType(codecFactory.SupportedMediaTypes(), mediaType)
	if !ok {
		sugar.Errorf("Can't get serializer")
	}
	encoder := codecFactory.EncoderForVersion(info.Serializer, gv)

	for {
		select {
		case <-closeChan:
			return
		case <-ticker.C:
			sugar.Infof("Listing pods in ns=%s with selector %#v", namespace, selector)
			list, err := kubeClient.Core().Pods(namespace).List(options)
			if err != nil {
				sugar.Errorf("Can't list %#v", err)
			}
			sugar.Infof("Listing returned %d items", len(list.Items))
			if response, err = runtime.Encode(encoder, list); err != nil {
				sugar.Errorf("Can't Encode list: %v", err)
			}
		}
	}
}

func getMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		// The "/" pattern matches everything, so we need to check
		// that we're at the root here.
		if req.URL.Path != "/" {
			http.NotFound(w, req)
			return
		}
		w.Write(response)
	})
	return mux
}

func runServer(addr string, mux http.Handler) {
	srv := http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		// We received an interrupt signal, shut down.
		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Printf("HTTP server Shutdown: %v", err)
		}
		close(closeChan)
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		// Error starting or closing listener:
		log.Printf("HTTP server ListenAndServe: %v", err)
	}

	<-closeChan

}

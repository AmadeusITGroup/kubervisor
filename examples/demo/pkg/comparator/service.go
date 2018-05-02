package comparator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/amadeusitgroup/kubervisor/examples/demo/pkg/api"
	"github.com/amadeusitgroup/kubervisor/examples/demo/pkg/message"
	"github.com/amadeusitgroup/kubervisor/examples/demo/pkg/utils"
)

// Service comparator service implementation
type Service struct {
	config *Config

	priceHistogram *prometheus.HistogramVec
}

// NewService returns new comparator Service instance
func NewService(cfg *Config) *Service {
	svc := &Service{
		config:         cfg,
		priceHistogram: utils.NewPriceHistogram(),
	}
	svc.init()
	return svc
}

// SearchHandler http handler func for multi proviser search flight solutions
func (s *Service) SearchHandler(w http.ResponseWriter, r *http.Request) {
	resp := message.Response{
		RequestInfo: message.Request{
			Date: time.Now(),
		},
	}
	if val, err := utils.GetParamValue(r, "origin"); err == nil {
		resp.RequestInfo.OD.Origin = api.CityCode(val)
	} else {
		resp.Errors = append(resp.Errors, api.Error{Code: int(http.StatusBadRequest), Description: fmt.Sprintf("%v", err)})
	}

	if val, err := utils.GetParamValue(r, "destination"); err == nil {
		resp.RequestInfo.OD.Destination = api.CityCode(val)
	} else {
		resp.Errors = append(resp.Errors, api.Error{Code: int(http.StatusBadRequest), Description: fmt.Sprintf("%v", err)})
	}

	for _, provider := range s.config.Providers {
		routes, err := getRoutesForProvider(provider, &resp.RequestInfo)
		if err != nil {
			resp.Errors = append(resp.Errors, api.Error{Code: int(http.StatusBadRequest), Description: fmt.Sprintf("%v", err)})
		}
		if routes != nil {
			for _, r := range routes {
				for _, flight := range r.Segments {
					s.priceHistogram.WithLabelValues(flight.Provider, flight.OD.String()).Observe(float64(flight.Price.Price))
					fmt.Println("Price:", flight.Price.Price)
				}
			}
			resp.Solutions = append(resp.Solutions, routes...)

		}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	if err := enc.Encode(resp); err != nil {
		fmt.Println("encode error:", err)
		http.Error(w, "unable ot encode response", http.StatusInternalServerError)
		return
	}
	returnCode := http.StatusOK
	for _, err := range resp.Errors {
		if err.Code > returnCode {
			returnCode = err.Code
		}
	}
	if returnCode != http.StatusOK {
		w.WriteHeader(returnCode)
	}
}

func (s *Service) init() {
	s.config.PromRegistry.Register(s.priceHistogram)
}

func getRoutesForProvider(provider string, req *message.Request) ([]api.Route, error) {
	resp, err := http.Get(getSearchURIForProvider(provider, string(req.OD.Origin), string(req.OD.Destination)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var response message.Response
	if err = json.Unmarshal(body, &response); err != nil {
		return nil, nil
	}

	return response.Solutions, nil
}

func getSearchURIForProvider(provider, origin, destination string) string {
	return fmt.Sprintf("http://%s/api/v1/search?origin=%s&destination=%s", provider, origin, destination)
}

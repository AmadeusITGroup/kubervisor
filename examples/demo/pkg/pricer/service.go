package pricer

import (
	"encoding/json"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/amadeusitgroup/kubervisor/examples/demo/pkg/api"
	"github.com/amadeusitgroup/kubervisor/examples/demo/pkg/generator"
	"github.com/amadeusitgroup/kubervisor/examples/demo/pkg/message"
	"github.com/amadeusitgroup/kubervisor/examples/demo/pkg/utils"
)

// Service pricer service struct
type Service struct {
	contigMutex sync.RWMutex
	config      *Config

	routeByODID    map[string][]api.Route
	priceHistogram *prometheus.HistogramVec
}

// NewService returns new Pricer service instance
func NewService(cfg *Config) *Service {
	svc := &Service{
		config:         cfg,
		priceHistogram: utils.NewPriceHistogram(),
	}
	svc.init()
	return svc
}

// SearchHandler HTTP Handler function for pricer search service
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

	conf := s.getConfig()
	if rs, ok := s.routeByODID[resp.RequestInfo.OD.String()]; ok {
		solutions := []api.Route{}
		for _, r := range rs {
			var solution api.Route
			solution = r
			for i, flight := range solution.Segments {
				price := generator.CalculateODPrice(conf.KmPrice, conf.RandPricePercent, flight.OD)
				solution.Segments[i].Price = api.Price{Price: price, Currency: "EUR"}
				s.priceHistogram.WithLabelValues(flight.Provider, flight.OD.String()).Observe(float64(price))
				fmt.Println("Price:", flight.Price.Price)
			}
			solutions = append(solutions, solution)
		}
		resp.Solutions = solutions
	} else {
		resp.Warnings = append(resp.Warnings, api.Warning{Description: fmt.Sprintf("No Solutions for OD: %s - %s ", resp.RequestInfo.OD.Origin, resp.RequestInfo.OD.Destination)})
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

func (s *Service) searchRoute(req *message.Request) ([]api.Route, error) {
	var routes []api.Route

	return routes, nil
}

// init Service initialisation
func (s *Service) init() {
	s.routeByODID = make(map[string][]api.Route)
	s.generateData()
	s.config.PromRegistry.Register(s.priceHistogram)
}

func (s *Service) generateData() {
	fmt.Println("- generate Data:")
	ods := generator.GetODs()
	now := time.Now()
	config := s.getConfig()
	for _, od := range ods {
		scheduldes := generator.GenerateSchedules(3, now)
		rs := generator.GenerateRoutes(scheduldes, config.Provider, config.KmPrice, config.RandPricePercent, []api.OriginDestination{od})
		s.routeByODID[od.String()] = rs
		fmt.Printf("  -> Add data for OD: %s-%s\n", od.Origin, od.Destination)
	}
}

// SetConfig HTTP Handler function to set a value in the config
func (s *Service) SetConfig(w http.ResponseWriter, r *http.Request) {
	newConfig := s.getConfig()
	for key, vals := range r.URL.Query() {
		switch key {
		case "kmprice":
			for _, val := range vals {
				if price, err := strconv.ParseFloat(val, 10); err == nil {
					newConfig.KmPrice = float32(price)
				}
			}
		}
	}
	s.setConfig(&newConfig)
}

func (s *Service) getConfig() Config {
	s.contigMutex.RLock()
	defer s.contigMutex.RUnlock()
	return *s.config
}

func (s *Service) setConfig(newConfig *Config) {
	s.contigMutex.Lock()
	defer s.contigMutex.Unlock()
	s.config = newConfig
}

package message

import (
	"github.com/amadeusitgroup/kubervisor/examples/demo/pkg/api"
)

// Response price service response
type Response struct {
	RequestInfo Request       `json:"request_info"`
	Solutions   []api.Route   `json:"solutions"`
	Errors      []api.Error   `json:"errors,omitempty"`
	Warnings    []api.Warning `json:"warnings,omitempty"`
}

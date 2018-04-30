package utils

import (
	"fmt"
	"net/http"
)

// GetParamValue return the value of a http parameter key
// if key didn't exist return an error
// if several iteration of the same key, return an error
func GetParamValue(r *http.Request, key string) (string, error) {
	keys, ok := r.URL.Query()[key]

	if ok && len(keys) == 1 {
		return r.URL.Query()[key][0], nil
	}

	if !ok {
		return "", fmt.Errorf("key: '%s' not found", key)
	} else {
		return "", fmt.Errorf("to many value for key, nb value: %d", len(keys))
	}
}

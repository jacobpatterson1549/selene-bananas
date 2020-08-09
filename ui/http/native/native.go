// Package native makes XML HTTP Requests using go code.
package native

import (
	"errors"
	net_http "net/http"

	"github.com/jacobpatterson1549/selene-bananas/ui/http"
)

// HTTPClient makes requests using the net/http package.
type HTTPClient struct {
	net_http.Client
}

// Do makes a HTTP request.
func (c HTTPClient) Do(req http.Request) (*http.Response, error) {
	httpRequest, err := net_http.NewRequest(req.Method, req.URL, req.Body)
	if err != nil {
		return nil, errors.New("could not create go http request: " + err.Error())
	}
	for k, v := range req.Headers {
		httpRequest.Header.Set(k, v)
	}
	httpResponse, err := c.Client.Do(httpRequest)
	if err != nil {
		return nil, errors.New("could not make http request: " + err.Error())
	}
	resp := http.Response{
		Code: httpResponse.StatusCode,
		Body: httpResponse.Body,
	}
	return &resp, nil
}

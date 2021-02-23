// +build js,wasm

// Package xhr makes XML HTTP Requests using native browser code.
package xhr

import (
	"errors"
	"io"
	"strings"
	"syscall/js"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/ui/dom"
	"github.com/jacobpatterson1549/selene-bananas/ui/http"
)

// HTTPClient makes requests using the net/http package.
// The request and response bodies are not streamed.
type HTTPClient struct {
	// Timeout is the amount of time a request can take before being considered timed out.
	Timeout time.Duration
}

// Do makes a HTTP request.
func (c HTTPClient) Do(req http.Request) (*http.Response, error) {
	xhr := dom.NewXHR()
	xhr.Call("open", req.Method, req.URL)
	timeoutMillis := c.Timeout.Milliseconds()
	xhr.Set("timeout", timeoutMillis)
	for k, v := range req.Headers {
		xhr.Call("setRequestHeader", k, v)
	}
	responseC := make(chan http.Response)
	errC := make(chan error)
	eventHandler := dom.NewJsEventFunc(handleEvent(xhr, responseC, errC))
	defer eventHandler.Release()
	for _, event := range []string{"load", "timeout", "abort"} {
		xhr.Call("addEventListener", event, eventHandler)
	}
	var body string
	if req.Body != nil {
		bytes, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, errors.New("getting request body: " + err.Error())
		}
		body = string(bytes)
	}
	xhr.Call("send", body)
	select {
	case response := <-responseC:
		return &response, nil
	case err := <-errC:
		return nil, err
	}
}

// handleEvent handles an event for the XHR.
func handleEvent(xhr js.Value, responseC chan<- http.Response, errC chan<- error) func(event js.Value) {
	return func(event js.Value) {
		eventType := event.Get("type").String()
		switch eventType {
		case "load":
			code := xhr.Get("status").Int()
			response := xhr.Get("response").String()
			responseR := strings.NewReader(response)
			body := io.NopCloser(responseR)
			responseC <- http.Response{
				Code: code,
				Body: body,
			}
		default:
			errC <- errors.New("received event type: " + eventType)
		}
	}
}

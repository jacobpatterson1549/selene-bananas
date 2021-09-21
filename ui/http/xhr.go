//go:build js && wasm

// Package http makes XML HTTP Requests using native browser code.
package http

import (
	"errors"
	"syscall/js"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/ui"
)

type (
	// Client makes HTTP requests.
	Client struct {
		// Timeout is the amount of time a request can take before being considered timed out.
		Timeout time.Duration
	}

	// Request identifies the question to ask a server.
	Request struct {
		// Method is the HTTP method (GET/POST).
		Method string
		// URL is the address to the server.
		URL string
		// Headers contain additional request properties.
		Headers map[string]string
		// Body contains additional request data.
		Body string
	}

	// Response is what the server responds.
	Response struct {
		// Code is a descriptive status about the server handled the response (200 OK, 500 Internal Server Error).
		Code int
		// Body contains the response data.
		Body string
		// err is used internally and is set if the response has an error.
		err error
	}
)

// Do makes a HTTP request.
func (c Client) Do(req Request) (*Response, error) {
	xhr := ui.NewXHR()
	xhr.Call("open", req.Method, req.URL)
	timeoutMillis := c.Timeout.Milliseconds()
	xhr.Set("timeout", timeoutMillis)
	for k, v := range req.Headers {
		xhr.Call("setRequestHeader", k, v)
	}
	responseC := make(chan Response)
	eventHandler := ui.NewJsEventFunc(handleEvent(xhr, responseC))
	defer eventHandler.Release()
	xhrEventTypes := []string{"load", "timeout", "abort", "error"}
	for _, event := range xhrEventTypes {
		xhr.Call("addEventListener", event, eventHandler)
	}
	go xhr.Call("send", req.Body)
	response := <-responseC
	if response.err != nil {
		return nil, response.err
	}
	return &response, nil
}

// handleEvent handles an event for the XHR.
func handleEvent(xhr js.Value, responseC chan<- Response) func(event js.Value) {
	return func(event js.Value) {
		eventType := event.Get("type").String()
		var r Response
		switch eventType {
		case "load":
			r.Code = xhr.Get("status").Int()
			r.Body = xhr.Get("response").String()
		default:
			r.err = errors.New("received event type: " + eventType)
		}
		responseC <- r
	}
}

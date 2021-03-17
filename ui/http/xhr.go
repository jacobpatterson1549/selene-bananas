// +build js,wasm

// Package http makes XML HTTP Requests using native browser code.
package http

import (
	"errors"
	"io"
	"strings"
	"syscall/js"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/ui/dom"
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
		Body io.Reader
	}

	// Response is what the server responds.
	Response struct {
		// Code is a descriptive status about the server handled the response (200 OK, 500 Internal Server Error).
		Code int
		// Body contains the response data.
		Body io.ReadCloser
	}
)

// Do makes a HTTP request.
func (c Client) Do(req Request) (*Response, error) {
	xhr := dom.NewXHR()
	xhr.Call("open", req.Method, req.URL)
	timeoutMillis := c.Timeout.Milliseconds()
	xhr.Set("timeout", timeoutMillis)
	for k, v := range req.Headers {
		xhr.Call("setRequestHeader", k, v)
	}
	responseC := make(chan Response)
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
func handleEvent(xhr js.Value, responseC chan<- Response, errC chan<- error) func(event js.Value) {
	return func(event js.Value) {
		eventType := event.Get("type").String()
		switch eventType {
		case "load":
			code := xhr.Get("status").Int()
			response := xhr.Get("response").String()
			responseR := strings.NewReader(response)
			body := io.NopCloser(responseR)
			responseC <- Response{
				Code: code,
				Body: body,
			}
		default:
			errC <- errors.New("received event type: " + eventType)
		}
	}
}

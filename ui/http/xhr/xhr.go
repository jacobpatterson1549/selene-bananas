// +build js,wasm

// Package xhr makes XML HTTP Requests using native browser code.
package xhr

import (
	"errors"
	"io/ioutil"
	"strings"
	"syscall/js"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/ui/dom"
	"github.com/jacobpatterson1549/selene-bananas/ui/http"
)

type (
	// HTTPClient makes requests using the net/http package.
	// The request and response bodies are not streamed.
	HTTPClient struct {
		// Timeout is the amount of time a request can take before being considered timed out.
		Timeout time.Duration
	}

	// response is passed between the XHR event handler and the request function
	response struct {
		resp *http.Response
		err  error
	}
)

// Do makes a HTTP request.
func (c HTTPClient) Do(req http.Request) (*http.Response, error) {
	xhr := dom.NewXHR()
	xhr.Call("open", req.Method, req.URL)
	timeoutMillis := c.Timeout.Milliseconds()
	xhr.Set("timeout", timeoutMillis)
	for k, v := range req.Headers {
		xhr.Call("setRequestHeader", k, v)
	}
	responseC := make(chan response)
	eventHandler := dom.NewJsEventFunc(handleEvent(xhr, responseC))
	for _, event := range []string{"load", "timeout", "abort"} {
		xhr.Call("addEventListener", event, eventHandler)
	}
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, errors.New("getting request body: " + err.Error())
	}
	xhr.Call("send", string(body))
	response := <-responseC
	eventHandler.Release() // TODO: add context support (if browser closed)
	if response.err != nil {
		return nil, response.err
	}
	return response.resp, nil
}

// handleEvent handles an event for the XHR.
func handleEvent(xhr js.Value, responseC chan<- response) func(event js.Value) {
	return func(event js.Value) {
		eventType := event.Get("type").String()
		var resp *response
		switch eventType {
		case "load":
			code := xhr.Get("status").Int()
			document := xhr.Get("response").String()
			documentR := strings.NewReader(document)
			body := ioutil.NopCloser(documentR)
			resp = &response{
				resp: &http.Response{
					Code: code,
					Body: body,
				},
			}
		default:
			resp = &response{
				err: errors.New("received event type: " + eventType),
			}
		}
		responseC <- *resp
	}
}

// Package http handles making http requests
package http

import "io"

type (
	// Client makes HTTP requests.
	Client interface {
		// Do makes a HTTP request.
		Do(Request) (*Response, error)
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

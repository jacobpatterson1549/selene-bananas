// +build js

package user

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/jacobpatterson1549/selene-bananas/go/ui/dom"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/log"
)

type (
	// Request contains the fields needed to make a request
	Request struct {
		Method    string
		URL       string
		URLSuffix string
		Params    url.Values
	}
)

var (
	httpClient       http.Client // TODO: add Timeout: 5 * time.Second,
	responseHandlers = map[string]func(r Request, body io.ReadCloser){
		"/user_create": func(r Request, b io.ReadCloser) {
			username := r.Params.Get("username")
			password := r.Params.Get("password")
			dom.StoreCredentials(username, password)
			Logout()
		},
		"/user_delete": func(r Request, body io.ReadCloser) {
			Logout()
		},
		"/user_login": func(r Request, body io.ReadCloser) {
			defer body.Close()
			jwt, err := ioutil.ReadAll(body)
			if err != nil {
				log.Error("reading response body: " + err.Error())
				return
			}
			username := r.Params.Get("username")
			password := r.Params.Get("password")
			dom.StoreCredentials(username, password)
			login(string(jwt))
		},
		"/user_update_password": func(r Request, body io.ReadCloser) {
			username := r.Params.Get("username")
			password := r.Params.Get("password_confirm")
			dom.StoreCredentials(username, password)
			Logout()
		},
		"/ping": func(r Request, body io.ReadCloser) {
			// NOOP
		},
	}
)

// Do makes a request to the server
// Requests should be made on separate goroutines.
func (r Request) Do() {
	if r.URLSuffix == "/user_delete" {
		message := "Are you sure? All accumulated points will be lost"
		if !dom.Confirm(message) {
			return
		}
	}
	rh, ok := responseHandlers[r.URLSuffix]
	if !ok {
		log.Error("Unknown action: " + r.URLSuffix)
		return
	}
	var httpRequest *http.Request
	var err error
	// TODO: pass context when making http request
	switch r.Method {
	case "get":
		httpRequest, err = http.NewRequest(r.Method, r.URL+"?"+r.Params.Encode(), nil)
	case "post":
		httpRequest, err = http.NewRequest(r.Method, r.URL, strings.NewReader(r.Params.Encode()))
		httpRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	default:
		log.Error("unknown method: " + r.Method)
		return
	}
	if err != nil {
		log.Error("creating request: " + err.Error())
		return
	}
	if dom.GetChecked("has-login") {
		jwt := JWT()
		httpRequest.Header.Set("Authorization", "Bearer "+jwt)
	}
	httpResponse, err := httpClient.Do(httpRequest)
	switch {
	case err != nil:
		log.Error("making http request: " + err.Error())
	case httpResponse.StatusCode >= 400:
		log.Error(httpResponse.Status)
		// TODO: logout user on http error?
	default:
		rh(r, httpResponse.Body)
	}
}

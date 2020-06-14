// +build js,wasm

package user

import (
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/jacobpatterson1549/selene-bananas/go/ui/dom"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/log"
)

var (
	httpClient       http.Client // TODO: add Timeout: 5 * time.Second, // TODO: make struct, keep this in the struct
	responseHandlers = map[string]func(f dom.Form, body io.ReadCloser){
		"/user_create": func(f dom.Form, b io.ReadCloser) {
			username := f.Params.Get("username")
			password := f.Params.Get("password")
			dom.StoreCredentials(username, password)
			Logout()
		},
		"/user_delete": func(f dom.Form, body io.ReadCloser) {
			Logout()
		},
		"/user_login": func(f dom.Form, body io.ReadCloser) {
			defer body.Close()
			jwt, err := ioutil.ReadAll(body)
			if err != nil {
				log.Error("reading response body: " + err.Error())
				return
			}
			username := f.Params.Get("username")
			password := f.Params.Get("password")
			dom.StoreCredentials(username, password)
			login(string(jwt))
		},
		"/user_update_password": func(f dom.Form, body io.ReadCloser) {
			username := f.Params.Get("username")
			password := f.Params.Get("password_confirm")
			dom.StoreCredentials(username, password)
			Logout()
		},
		"/ping": func(f dom.Form, body io.ReadCloser) {
			// NOOP
		},
	}
)

// Request makes a request to the server using the fields in the form.
func request(f dom.Form) {
	if f.URLSuffix == "/user_delete" {
		message := "Are you sure? All accumulated points will be lost"
		if !dom.Confirm(message) {
			return
		}
	}
	rh, ok := responseHandlers[f.URLSuffix]
	if !ok {
		log.Error("Unknown action: " + f.URLSuffix)
		return
	}
	var httpRequest *http.Request
	var err error
	// TODO: pass context when making http request
	switch f.Method {
	case "get":
		httpRequest, err = http.NewRequest(f.Method, f.URL+"?"+f.Params.Encode(), nil)
	case "post":
		httpRequest, err = http.NewRequest(f.Method, f.URL, strings.NewReader(f.Params.Encode()))
		httpRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	default:
		log.Error("unknown method: " + f.Method)
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
		rh(f, httpResponse.Body)
	}
}

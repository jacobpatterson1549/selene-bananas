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

// Request makes a request to the server using the fields in the form.
func (u User) request(f dom.Form) {
	if f.URL.Path == "/user_delete" { // TODO: make responseHandler return a validation func AND response handler, for user_delete and user_create/modify additional username/password validations.
		message := "Are you sure? All accumulated points will be lost"
		if !dom.Confirm(message) {
			return
		}
	}
	rh := u.responseHandler(f.URL.Path, f)
	if rh == nil {
		log.Error("Unknown action: " + f.URL.Path)
		return
	}
	var httpRequest *http.Request
	var err error
	// TODO: pass context when making http request
	switch f.Method {
	case "get":
		f.URL.RawQuery = f.Params.Encode()
		url := f.URL.String()
		httpRequest, err = http.NewRequest(f.Method, url, nil)
	case "post":
		url := f.URL.String()
		body := strings.NewReader(f.Params.Encode())
		httpRequest, err = http.NewRequest(f.Method, url, body)
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
		jwt := u.JWT()
		httpRequest.Header.Set("Authorization", "Bearer "+jwt)
	}
	httpResponse, err := u.httpClient.Do(httpRequest)
	switch {
	case err != nil:
		log.Error("making http request: " + err.Error())
		return
	case httpResponse.StatusCode >= 400:
		log.Error(httpResponse.Status)
		// TODO: logout user on http error?
		return
	default:
		rh(httpResponse.Body)
	}
}

// responseHandler creates a response-handling function for the url and form.  Nil is returned if the url is unknown.Path
func (u *User) responseHandler(urlPath string, f dom.Form) func(body io.ReadCloser) {
	switch urlPath {
	case "/user_create":
		return func(body io.ReadCloser) {
			username := f.Params.Get("username")
			password := f.Params.Get("password")
			dom.StoreCredentials(username, password)
			u.Logout()
		}
	case "/user_delete":
		return func(body io.ReadCloser) {
			u.Logout()
		}
	case "/user_login":
		return func(body io.ReadCloser) {
			defer body.Close()
			jwt, err := ioutil.ReadAll(body)
			if err != nil {
				log.Error("reading response body: " + err.Error())
				return
			}
			username := f.Params.Get("username")
			password := f.Params.Get("password")
			dom.StoreCredentials(username, password)
			u.login(string(jwt))
		}
	case "/user_update_password":
		return func(body io.ReadCloser) {
			username := f.Params.Get("username")
			password := f.Params.Get("password_confirm")
			dom.StoreCredentials(username, password)
			u.Logout()
		}
	case "/ping":
		return func(body io.ReadCloser) {
			// NOOP
		}
	default:
		return nil
	}
}

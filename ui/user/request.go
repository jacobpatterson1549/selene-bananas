// +build js,wasm

package user

import (
	"errors"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/ui/dom"
	"github.com/jacobpatterson1549/selene-bananas/ui/http"
)

// request handles http communication with the server through forms.
type request struct {
	user      *User
	form      dom.Form
	validator func() bool
	handler   func(body io.ReadCloser)
}

// Request makes an BLOCKING request to the server using the fields in the form.
func (u *User) request(event js.Value) {
	f, err := dom.NewForm(event)
	if err != nil {
		u.log.Error(err.Error())
		return
	}
	r, err := u.newRequest(*f)
	switch {
	case err != nil:
		u.log.Error(err.Error())
		return
	case r.validator != nil:
		if ok := r.validator(); !ok {
			return
		}
	}
	resp, err := r.do()
	switch {
	case err != nil:
		u.log.Error("making http request: " + err.Error())
		return
	case resp.Code >= 400:
		u.handleResponseError(resp)
		return
	case r.handler != nil:
		r.handler(resp.Body)
	}
}

// do actually makes the request.
func (r request) do() (*http.Response, error) {
	f := r.form
	req := http.Request{
		Method:  f.Method,
		Headers: make(map[string]string),
	}
	switch f.Method {
	case "get":
		f.URL.RawQuery = f.Params.Encode()
		req.URL = f.URL.String()
	case "post":
		req.URL = f.URL.String()
		req.Body = strings.NewReader(f.Params.Encode())
		req.Headers["Content-Type"] = "application/x-www-form-urlencoded"
	default:
		return nil, errors.New("unknown method: " + f.Method)
	}
	if dom.Checked("#has-login") {
		jwt := r.user.JWT()
		req.Headers["Authorization"] = "Bearer " + jwt
	}
	return r.user.httpClient.Do(req)
}

// responseHandler creates a response-handling function for the url and form.  Nil is returned if the url is unknown.Path
func (u *User) newRequest(f dom.Form) (*request, error) {
	var validator func() bool
	var handler func(body io.ReadCloser)
	switch f.URL.Path {
	case "/user_create", "/user_update_password":
		handler = func(body io.ReadCloser) {
			f.StoreCredentials()
			u.Logout()
		}
	case "/user_delete":
		validator = func() bool {
			message := "Are you sure? All accumulated points will be lost"
			ok := dom.Confirm(message)
			return ok
		}
		handler = func(body io.ReadCloser) {
			u.Logout()
		}
	case "/user_login":
		handler = func(body io.ReadCloser) {
			defer body.Close()
			jwt, err := ioutil.ReadAll(body)
			if err != nil {
				u.log.Error("reading response body: " + err.Error())
				return
			}
			f.StoreCredentials()
			u.login(string(jwt))
		}
	case "/ping":
		// NOOP
	default:
		return nil, errors.New("Unknown action: " + f.URL.Path)
	}
	r := request{
		user:      u,
		form:      f,
		validator: validator,
		handler:   handler,
	}
	return &r, nil
}

func (u *User) handleResponseError(resp *http.Response) {
	u.Logout()
	body := resp.Body
	defer body.Close()
	message, err := ioutil.ReadAll(body)
	switch {
	case err != nil:
		u.log.Error("reading error response body: " + err.Error())
	case len(message) > 0:
		u.log.Error("HTTP error: status " + strconv.Itoa(resp.Code) + ": " + string(message))
	}
}

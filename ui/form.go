//go:build js && wasm

package ui

import (
	"errors"
	"syscall/js"
)

// Form contains the fields needed to make a request to the server.
type Form struct {
	v      js.Value
	Method string
	URL    URL
	Params Values
}

// NewForm creates a form from the target property of the event.  An error is returned if the url action is not successfully parsed.
func NewForm(event js.Value) (*Form, error) {
	var dom DOM
	form := event.Get("target")
	method := form.Get("method").String()
	action := form.Get("action").String()
	u, err := Parse(action)
	if err != nil {
		return nil, errors.New("getting url from form action: " + err.Error())
	}
	formInputs := dom.QuerySelectorAll(form, `input[name]:not([type="submit"])`)
	params := make(Values, len(formInputs))
	for _, formInput := range formInputs {
		name := formInput.Get("name").String()
		value := formInput.Get("value").String()
		params.Add(name, value)
	}
	f := Form{
		v:      form,
		Method: method,
		URL:    *u,
		Params: params,
	}
	return &f, nil
}

// Reset clears the named inputs of the form.
func (f *Form) Reset() {
	var dom DOM
	formInputs := dom.QuerySelectorAll(f.v, `input[name]:not([type="submit"])`)
	for _, formInput := range formInputs {
		formInput.Set("value", "")
	}
}

// StoreCredentials attempts to save the credentials for the login, if browser wants to.
func (f *Form) StoreCredentials() {
	global := js.Global()
	passwordCredential := global.Get("PasswordCredential")
	if passwordCredential.Truthy() {
		c := passwordCredential.New(f.v)
		navigator := global.Get("navigator")
		credentials := navigator.Get("credentials")
		credentials.Call("store", c)
	}
}

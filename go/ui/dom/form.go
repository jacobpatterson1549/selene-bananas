// +build js,wasm

package dom

import (
	"errors"
	"net/url"
	"syscall/js"
)

type (
	// Form contains the fields needed to make a request to the server.
	Form struct {
		v      js.Value
		Method string
		URL    url.URL
		Params url.Values
	}
)

func NewForm(event js.Value) (*Form, error) {
	form := event.Get("target")
	method := form.Get("method").String()
	action := form.Get("action").String()
	url, err := url.Parse(action)
	if err != nil {
		return nil, errors.New("getting url from form action: " + err.Error())
	}
	formInputs := form.Call("querySelectorAll", `input[name]:not([type="submit"])`)
	params := make(map[string][]string, formInputs.Length())
	for i := 0; i < formInputs.Length(); i++ {
		formInput := formInputs.Index(i)
		name := formInput.Get("name").String()
		value := formInput.Get("value").String()
		params[name] = []string{value}
	}
	f := Form{
		v:      form,
		Method: method,
		URL:    *url,
		Params: params,
	}
	return &f, nil
}

// Reset clears the named inputs of the form.
func (f *Form) Reset() {
	formInputs := f.v.Call("querySelectorAll", `input[name]:not([type="submit"])`)
	for i := 0; i < formInputs.Length(); i++ {
		formInput := formInputs.Index(i)
		formInput.Set("value", "")
	}
}

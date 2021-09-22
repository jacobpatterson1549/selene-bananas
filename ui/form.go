//go:build js && wasm

package ui

import (
	"errors"
	"syscall/js"
)

type (
	// Form contains the fields needed to make a request to the server.
	Form struct {
		Element js.Value
		Method  string
		URL     URL
		Params  Values
		querier Querier
	}

	// Querier selects on the form.
	Querier func(form js.Value, query string) []js.Value
)

// NewForm creates a form from the target property of the event.  An error is returned if the url action is not successfully parsed.
func NewForm(querier Querier, event js.Value) (*Form, error) {
	if querier == nil {
		return nil, errors.New("no querier specified")
	}
	form := event.Get("target")
	method := form.Get("method").String()
	action := form.Get("action").String()
	u, err := Parse(action)
	if err != nil {
		return nil, errors.New("getting url from form action: " + err.Error())
	}
	f := Form{
		Element: form,
		Method:  method,
		URL:     *u,
		querier: querier,
	}
	formInputs := f.NonSubmitInputs()
	f.Params = make(Values, len(formInputs))
	for _, formInput := range formInputs {
		name := formInput.Get("name").String()
		value := formInput.Get("value").String()
		f.Params.Add(name, value)
	}
	return &f, nil
}

// Reset clears the named inputs of the form.
func (f *Form) Reset() {
	formInputs := f.querier(f.Element, `input[name]:not([type="submit"])`)
	for _, formInput := range formInputs {
		formInput.Set("value", "")
	}
}

// NonSubmitInputs uses the querier to get named inputs on the form that do not have a type of submit.
func (f Form) NonSubmitInputs() []js.Value {
	return f.querier(f.Element, `input[name]:not([type="submit"])`)
}

//go:build js && wasm

// Package ui contains the client for the game.
// It compiles to webassembly to allow users to play the game in their browsers.
// Package dom contains the javascript bindings for the site
package ui

import (
	"syscall/js"
	"time"
)

// DOM represents common functions on the Document Object Model.
type DOM struct{}

// QuerySelector returns the first element returned by the query from root of the document.
func (dom DOM) QuerySelector(query string) js.Value {
	global := js.Global()
	document := global.Get("document")
	return document.Call("querySelector", query)
}

// QuerySelectorAll returns an array of the elements returned by the query from the specified document.
func (dom DOM) QuerySelectorAll(document js.Value, query string) []js.Value {
	value := document.Call("querySelectorAll", query)
	values := make([]js.Value, value.Length())
	for i := 0; i < len(values); i++ {
		values[i] = value.Index(i)
	}
	return values
}

// Checked returns whether the element has a checked value of true.
func (dom DOM) Checked(query string) bool {
	element := dom.QuerySelector(query)
	checked := element.Get("checked")
	return checked.Bool()
}

// SetChecked sets the checked property of the element.
func (dom *DOM) SetChecked(query string, checked bool) {
	element := dom.QuerySelector(query)
	element.Set("checked", checked)
}

// Value gets the value of the input element.
func (dom DOM) Value(query string) string {
	element := dom.QuerySelector(query)
	value := element.Get("value")
	return value.String()
}

// SetValue sets the value of the input element.
func (dom *DOM) SetValue(query, value string) {
	element := dom.QuerySelector(query)
	element.Set("value", value)
}

// SetButtonDisabled sets the disable property of the button element.
func (dom *DOM) SetButtonDisabled(query string, disabled bool) {
	element := dom.QuerySelector(query)
	element.Set("disabled", disabled)
}

// FormatTime formats a datetime to HH:MM:SS.
func (dom DOM) FormatTime(utcSeconds int64) string {
	t := time.Unix(utcSeconds, 0).Local() // uses local timezone
	return t.Format("15:04:05")
}

// CloneElement creates a close of the element, which should be a template.
func (dom DOM) CloneElement(query string) js.Value {
	templateElement := dom.QuerySelector(query)
	contentElement := templateElement.Get("content")
	clone := contentElement.Call("cloneNode", true)
	return clone
}

// Confirm shows a popup asking the user a yes/no question.
// The true return value implies the "yes" choice.
func (dom *DOM) Confirm(message string) bool {
	global := js.Global()
	result := global.Call("confirm", message)
	return result.Bool()
}

// alert shows a popup in the browser.
func (dom *DOM) alert(message string) {
	global := js.Global()
	global.Call("alert", message)
}

// Color returns the text color of the element after css has been applied.
func (dom DOM) Color(element js.Value) string {
	global := js.Global()
	computedStyle := global.Call("getComputedStyle", element)
	color := computedStyle.Get("color")
	return color.String()
}

// NewWebSocket creates a new WebSocket with the specified url.
func (dom *DOM) NewWebSocket(url string) js.Value {
	global := js.Global()
	webSocket := global.Get("WebSocket")
	return webSocket.New(url)
}

// NewXHR creates a new XML HTTP Request.
func (dom *DOM) NewXHR() js.Value {
	global := js.Global()
	xhr := global.Get("XMLHttpRequest")
	return xhr.New()
}

// Base64Decode decodes the ascii base-64 string to binary (atob).
// Panics if the encodedData is not a valid url encoded base64 string.
func (dom DOM) Base64Decode(a string) []byte {
	global := js.Global()
	s := global.Call("atob", a)
	return []byte(s.String())
}

// StoreCredentials attempts to save the credentials for the login, if browser wants to.
func (dom *DOM) StoreCredentials(form js.Value) {
	global := js.Global()
	passwordCredential := global.Get("PasswordCredential")
	if passwordCredential.Truthy() {
		c := passwordCredential.New(form)
		navigator := global.Get("navigator")
		credentials := navigator.Get("credentials")
		credentials.Call("store", c)
	}
}

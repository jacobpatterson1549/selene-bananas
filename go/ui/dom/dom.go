// +build js,wasm

// Package dom contains the javascript bindings for the site
package dom

import (
	"syscall/js"
	"time"
)

// QuerySelector returns the first element returned by the query from root of the document.
func QuerySelector(query string) js.Value {
	global := js.Global()
	document := global.Get("document")
	return document.Call("querySelector", query)
}

// QuerySelectorAll returns an array of the elements returned by the query from the specified document.
func QuerySelectorAll(document js.Value, query string) js.Value {
	return document.Call("querySelectorAll", query)
}

// SetCheckedQuery sets the checked property of the element.
func SetCheckedQuery(query string, checked bool) {
	element := QuerySelector(query)
	element.Set("checked", checked)
}

// GetCheckedQuery returns whether the element has a checked value of true.
func GetCheckedQuery(query string) bool {
	element := QuerySelector(query)
	checked := element.Get("checked")
	return checked.Bool()
}

// GetValue gets the value of the input element.
func GetValue(query string) string {
	element := QuerySelector(query)
	value := element.Get("value")
	return value.String()
}

// SetValue sets the value of the input element.
func SetValue(query, value string) {
	element := QuerySelector(query)
	element.Set("value", value)
}

// SetButtonDisabled sets the disable property of the button element.
func SetButtonDisabled(query string, disabled bool) {
	element := QuerySelector(query)
	element.Set("disabled", disabled)
}

// FormatTime formats a datetime to HH:MM:SS.
func FormatTime(utcSeconds int64) string {
	t := time.Unix(utcSeconds, 0) // uses local timezone
	return t.Format("15:04:05")
}

// CloneElement creates a close of the element, which should be a template.
func CloneElement(query string) js.Value {
	templateElement := QuerySelector(query)
	contentElement := templateElement.Get("content")
	clone := contentElement.Call("cloneNode", true)
	return clone
}

// Confirm shows a popup asking the user a yes/no question.
// The true return value implies the "yes" choice.
func Confirm(message string) bool {
	global := js.Global()
	result := global.Call("confirm", message)
	return result.Bool()
}

// NewWebSocket creates a new WebSocket with the specified url.
func NewWebSocket(url string) js.Value {
	global := js.Global()
	return global.Get("WebSocket").New(url)
}

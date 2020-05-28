// +build js

// Package js contains the javascript bindings for the site
package js

import (
	syscall_js "syscall/js"
)

var document syscall_js.Value = syscall_js.Global().Get("document")

func getElementById(id string) syscall_js.Value {
	return document.Call("getElementById", id)
}

// SetChecked sets the checked property of the element with the specified element id
func SetChecked(id string, checked bool) {
	element := getElementById(id)
	element.Set("checked", checked)
}

// GetChecked returns whether the element has a checked value of true.
func GetChecked(id string) bool {
	element := getElementById(id)
	checked := element.Get("checked")
	return checked.Bool()
}

// SetInnerHTML sets the inner html of the element with the specified id.
func SetInnerHTML(id string, innerHTML string) {
	element := getElementById(id)
	element.Set("innerHTML", innerHTML)
}

// GetValue gets the value of the input element with the specified id.
func GetValue(id string) string {
	element := getElementById(id)
	value := element.Get("value")
	return value.String()

}

// SetValue sets the value of the input element with the specified id.
func SetValue(id, value string) {
	element := getElementById(id)
	element.Set("value", value)
}

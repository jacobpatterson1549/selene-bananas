// +build js

// Package js contains the javascript bindings for the site
package js

import (
	"syscall/js"
	"time"
)

var document js.Value = js.Global().Get("document")

func getElementById(id string) js.Value {
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

// DormatDate formats a datetime to HH:MM:SS.
func FormatDate(time time.Time) string {
	return time.Format("15:04:05")
}

// AddLog adds a log message with the specified class
func AddLog(class, text string) {
	logItemTemplate := getElementById("log-item")
	logItemTemplateContent := logItemTemplate.Get("content")
	clone := logItemTemplateContent.Call("cloneNode", true)
	cloneChildren := clone.Get("children")
	logItemElement := cloneChildren.Index(0)
	time := FormatDate(time.Now())
	textContent := time + " : " + text
	logItemElement.Set("textContent", textContent)
	logItemElement.Set("className", class)
	logScrollElement := getElementById("log-scroll")
	logScrollElement.Call("appendChild", logItemElement)
	scrollHeight := logScrollElement.Get("scrollHeight")
	clientHeight := logScrollElement.Get("clientHeight")
	scrollTop := scrollHeight.Int() - clientHeight.Int()
	logScrollElement.Set("scrollTop", scrollTop)
}

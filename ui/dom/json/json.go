// +build js,wasm

// Package json uses the dom JSON object for encoding/decoding.
package json

import "syscall/js"

var json = js.Global().Get("JSON")

// Parse converts the text into a JS Value.
func Parse(text string) js.Value {
	return json.Call("parse", text)
}

// Stringify converts the value into a JSON string.
func Stringify(value interface{}) string {
	return json.Call("stringify", value).String()
}

// +build js,wasm

// Package json uses the dom JSON object for encoding/decoding.
package json

import (
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/ui/dom"
)

var json = js.Global().Get("JSON")

// Parse converts the text into a JS Value.
func Parse(text string) (value *js.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = dom.RecoverError(r)
		}
	}()
	v := json.Call("parse", text)
	return &v, nil
}

// Stringify converts the value into a JSON string.
func Stringify(value interface{}) (text string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = dom.RecoverError(r)
		}
	}()
	j := json.Call("stringify", value).String()
	return j, nil
}

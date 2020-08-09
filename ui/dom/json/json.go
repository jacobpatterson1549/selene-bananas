// +build js,wasm

// Package json uses the dom JSON object for encoding/decoding.
package json

import (
	"errors"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/ui/dom"
)

var json = js.Global().Get("JSON")

// Parse converts the text into a JS Value.
func Parse(data string, v interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = dom.RecoverError(r)
			err = errors.New("JSON parse: " + err.Error())
		}
	}()
	jsValue := json.Call("parse", data)
	return fromMap(v, jsValue)
}

// Stringify converts the value into a JSON string.
func Stringify(value interface{}) (text string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = dom.RecoverError(r)
			err = errors.New("JSON stringify: " + err.Error())
		}
	}()
	j := json.Call("stringify", value).String()
	return j, nil
}

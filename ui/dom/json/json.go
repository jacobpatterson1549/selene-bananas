// +build js,wasm

// Package json uses the dom JSON object for encoding/decoding.
package json

import (
	"errors"
	"strconv"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/ui/dom"
)

var json = js.Global().Get("JSON")
var array = js.Global().Get("Array")
var object = js.Global().Get("Object")

// Parse converts the text into a JS Value.
func Parse(data string, v interface{}) (err error) {
	defer func() {
		if r := recover(); err == nil && r != nil {
			err = dom.RecoverError(r)
			err = errors.New("JSON parse: " + err.Error())
		}
	}()
	jsValue := json.Call("parse", data)
	i, err := toInterface(jsValue)
	if err != nil {
		return errors.New("converting json js value to interface: " + err.Error())
	}
	return fromObject(v, i)
}

// Stringify converts the value into a JSON string.
func Stringify(value interface{}) (text string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = dom.RecoverError(r)
			err = errors.New("JSON stringify: " + err.Error())
		}
	}()
	o, err := toObject(value)
	if err != nil {
		return "", err
	}
	v := json.Call("stringify", o)
	return v.String(), nil
}

// toInterface converts the jsValue into an interface.  It can be thought of an the opposite of js.ValueOf(interface{}).
func toInterface(jsValue js.Value) (interface{}, error) {
	if jsValue.IsUndefined() || jsValue.IsNull() {
		return nil, nil
	}
	switch t := jsValue.Type(); t {
	case js.TypeString:
		return jsValue.String(), nil
	case js.TypeNumber:
		return jsValue.Int(), nil
	case js.TypeBoolean:
		return jsValue.Bool(), nil
	case js.TypeObject:
		if jsValue.InstanceOf(array) {
			return toArray(jsValue)
		}
		return toMap(jsValue)
	default:
		return nil, errors.New("unknown type: " + t.String())
	}
}

// toArray converts the jsValue that is an arry int o a slice of interfaces.
func toArray(jsValue js.Value) ([]interface{}, error) {
	n := jsValue.Length()
	if n == 0 {
		return nil, nil
	}
	a := make([]interface{}, n)
	for i := 0; i < n; i++ {
		v := jsValue.Index(i)
		o, err := toInterface(v)
		if err != nil {
			return nil, errors.New("index " + strconv.Itoa(i) + ": " + err.Error())
		}
		a[i] = o
	}
	return a, nil
}

// toMap converts the jsValue that is an object into a map of strings to interfaces.
func toMap(jsValue js.Value) (map[string]interface{}, error) {
	keys := object.Call("keys", jsValue)
	properties, err := toArray(keys)
	n := len(properties)
	if n == 0 {
		return nil, nil
	}
	m := make(map[string]interface{}, n)
	switch {
	case err != nil:
		return nil, errors.New("getting object keys: " + err.Error())
	case len(properties) != n:
		return nil, errors.New("wanted " + strconv.Itoa(n) + " keys, got " + strconv.Itoa(len(properties)))
	}
	for i := 0; i < n; i++ {
		k, ok := properties[i].(string)
		if !ok {
			return nil, errors.New("key at index " + strconv.Itoa(i) + " is not a string")
		}
		v := jsValue.Get(k)
		o, err := toInterface(v)
		if err != nil {
			return nil, errors.New("index " + strconv.Itoa(i) + ": " + err.Error())
		}
		m[k] = o
	}
	return m, nil
}

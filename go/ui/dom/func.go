// +build js,wasm

package dom

import (
	"syscall/js"
)

// RegisterFunc sets the function as a field on the parent.
// The parent object is created if it does not exist.
func RegisterFunc(parentName, fnName string, fn js.Func) {
	parent := js.Global().Get(parentName)
	if parent.IsUndefined() {
		parent = js.ValueOf(make(map[string]interface{}))
		js.Global().Set(parentName, parent)
	}
	parent.Set(fnName, fn)
}

// RegisterEventListenerFunc adds an event listener to the parent that processes an event and returns it as a javascript function
func RegisterEventListenerFunc(parent js.Value, fnName string, fn func(event js.Value)) js.Func {
	jsFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		fn(event)
		return nil
	})
	args := []interface{}{
		fnName,
		jsFunc,
		map[string]interface{}{
			"passive": false,
		},
	}
	parent.Call("addEventListener", args...)
	return jsFunc
}

// NewJsFunc creates a new javascript function from the provided function.
func NewJsFunc(fn func()) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		fn()
		return nil
	})
}

// NewJsEventFunc creates a new javascript function from the provided function that processes an event and returns nothing.
// PreventDefault is called on the event before applying the function
func NewJsEventFunc(fn func(event js.Value)) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		event.Call("preventDefault")
		fn(event)
		return nil
	})
}

// NewJsStringFunc creates a new javascript function that has no inputs and returns a string.
func NewJsStringFunc(fn func() string) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return fn()
	})
}

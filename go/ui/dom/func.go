// +build js,wasm

package dom

import (
	"syscall/js"
)

// RegisterFunc sets the function as a field on the parent.
// The parent object is create if it does not exist.
func RegisterFunc(parentName, fnName string, fn js.Func) {
	parent := js.Global().Get(parentName)
	if parent.IsUndefined() {
		parent = js.ValueOf(make(map[string]interface{}))
		js.Global().Set(parentName, parent)
	}
	parent.Set(fnName, fn)
}

// NewJsFunc creates a new javascript function from the provided function.
func NewJsFunc(fn func()) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		fn()
		return nil
	})
}

// NewJsFuncEvent creates a new javascript function from the provided function that processes an event and returns nothing.
// PreventDefault is called on the event before applying the function
func NewJsFuncEvent(fn func(event js.Value)) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		event.Call("preventDefault")
		fn(event)
		return nil
	})
}

// NewJsFuncString creates a new javascript function that has no inputs and returns a string.
func NewJsFuncString(fn func() string) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return fn()
	})
}

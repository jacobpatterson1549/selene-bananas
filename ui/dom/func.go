// +build js,wasm

package dom

import (
	"context"
	"sync"
	"syscall/js"
)

// RegisterFuncs sets the function as fields on the parent.
// The parent object is created if it does not exist.
func RegisterFuncs(ctx context.Context, wg *sync.WaitGroup, parentName string, jsFuncs map[string]js.Func) {
	global := js.Global()
	parent := global.Get(parentName)
	if parent.IsUndefined() {
		parent = js.ValueOf(make(map[string]interface{}))
		global.Set(parentName, parent)
	}
	for fnName, fn := range jsFuncs {
		parent.Set(fnName, fn)
	}
	wg.Add(1)
	go ReleaseJsFuncsOnDone(ctx, wg, jsFuncs)
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
	return NewJsEventFuncAsync(fn, false)
}

// NewJsEventFuncAsync performs similarly to NewJsEventFunc, but calls the event-handling function asynchronously if async is true.
func NewJsEventFuncAsync(fn func(event js.Value), async bool) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		event.Call("preventDefault")
		switch {
		case async:
			go fn(event)
		default:
			fn(event)
		}
		return nil
	})
}

// ReleaseJsFuncsOnDone releases the jsFuncs and decrements the waitgroup when the context is done.
// This function should be called on a separate goroutine.
func ReleaseJsFuncsOnDone(ctx context.Context, wg *sync.WaitGroup, jsFuncs map[string]js.Func) {
	<-ctx.Done() // BLOCKING
	for _, f := range jsFuncs {
		f.Release()
	}
	wg.Done()
}

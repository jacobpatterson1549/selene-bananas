//go:build js && wasm

package ui

import (
	"context"
	"errors"
	"strings"
	"sync"
	"syscall/js"
)

// RegisterFuncs sets the function as fields on the parent.
// The parent object is created if it does not exist.
func (dom *DOM) RegisterFuncs(ctx context.Context, wg *sync.WaitGroup, parentName string, jsFuncs map[string]js.Func) {
	parent := dom.global.Get(parentName)
	if parent.IsUndefined() {
		parent = js.ValueOf(make(map[string]interface{}))
		dom.global.Set(parentName, parent)
	}
	for fnName, fn := range jsFuncs {
		parent.Set(fnName, fn)
	}
	wg.Add(1)
	go dom.ReleaseJsFuncsOnDone(ctx, wg, jsFuncs)
}

// NewJsFunc creates a new javascript function from the provided function.
func (dom *DOM) NewJsFunc(fn func()) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer dom.AlertOnPanic()
		fn()
		return nil
	})
}

// NewJsEventFunc creates a new javascript function from the provided function that processes an event and returns nothing.
// PreventDefault is called on the event before applying the function
func (dom *DOM) NewJsEventFunc(fn func(event js.Value)) js.Func {
	return dom.NewJsEventFuncAsync(fn, false)
}

// NewJsEventFuncAsync performs similarly to NewJsEventFunc, but calls the event-handling function asynchronously if async is true.
func (dom *DOM) NewJsEventFuncAsync(fn func(event js.Value), async bool) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		event.Call("preventDefault")
		runFn := func() {
			defer dom.AlertOnPanic()
			fn(event)
		}
		switch {
		case async:
			go runFn()
		default:
			runFn()
		}
		return nil
	})
}

// ReleaseJsFuncsOnDone releases the jsFuncs and decrements the waitgroup when the context is done.
// This function should be called on a separate goroutine.
func (dom *DOM) ReleaseJsFuncsOnDone(ctx context.Context, wg *sync.WaitGroup, jsFuncs map[string]js.Func) {
	defer dom.AlertOnPanic()
	<-ctx.Done() // BLOCKING
	for _, f := range jsFuncs {
		f.Release()
	}
	wg.Done()
}

// AlertOnPanic checks to see if a panic has occurred.
// This function should be deferred as the first statement for each goroutine.
func (dom *DOM) AlertOnPanic() {
	if r := recover(); r != nil {
		err := dom.recoverError(r)
		f := []string{
			"FATAL: site shutting down",
			"See browser console for more information",
			"Message: " + err.Error(),
		}
		message := strings.Join(f, "\n")
		dom.alert(message)
		panic(err)
	}
}

// RecoverError converts the recovery interface into a useful error.
// Panics if the interface is not an error or a string.
func (dom *DOM) recoverError(r interface{}) error {
	switch v := r.(type) {
	case error:
		return v
	case string:
		return errors.New(v)
	default:
		panic([]interface{}{"unknown panic type", v, r})
	}
}

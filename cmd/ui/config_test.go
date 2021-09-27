//go:build js && wasm

package main

import (
	"context"
	"sync"
	"syscall/js"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/ui"
)

func TestInitDom(t *testing.T) {
	global := js.ValueOf(map[string]interface{}{}) // a mock global value that points to inself
	jsFuncs := map[string]js.Func{
		"querySelector":    js.FuncOf(func(this js.Value, args []js.Value) interface{} { return global }),
		"getContext":       js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }),
		"getComputedStyle": js.FuncOf(func(this js.Value, args []js.Value) interface{} { return global }),
		"addEventListener": js.FuncOf(func(this js.Value, args []js.Value) interface{} { return global }),
	}
	global.Set("document", global)
	f := flags{
		dom: ui.NewDOM(global),
	}
	for name, jsFunc := range jsFuncs {
		global.Set(name, jsFunc)
	}
	ctx := context.Background()
	ctx, cancelFunc := context.WithCancel(ctx)
	var wg sync.WaitGroup
	f.initDom(ctx, &wg)
	cancelFunc()
	wg.Wait()
	for _, jsFunc := range jsFuncs {
		jsFunc.Release()
	}
	wantComponents := []string{
		"log",
		"user",
		"game",
		"lobby",
	}
	for _, want := range wantComponents {
		c := global.Get(want)
		if !c.Truthy() {
			t.Errorf("wanted component %v to be set on global", want)
		}
	}
}

//go:build js && wasm

// Package main initializes interactive frontend elements and runs as long as the webpage is open.
package main

import (
	"context"
	"sync"
	"syscall/js"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/ui"
)

// main initializes the wasm code for the web dom and runs as long as the browser is open.
func main() {
	global := js.Global()
	dom := ui.NewDOM(global)
	defer dom.AlertOnPanic()
	f := flags{
		dom:         dom,
		httpTimeout: 10 * time.Second,
		tileLength:  25, // also in game.html
	}
	ctx := context.Background()
	ctx, cancelFunc := context.WithCancel(ctx)
	var wg sync.WaitGroup
	f.initDom(ctx, &wg)
	enableInteraction(*dom)
	initBeforeUnloadFn(cancelFunc, &wg, global)
	wg.Wait() // BLOCKING
}

// initBeforeUnloadFn registers a function to cancel the context when the browser is about to close.
// This should trigger other dom functions to release.
func initBeforeUnloadFn(cancelFunc context.CancelFunc, wg *sync.WaitGroup, global js.Value) {
	wg.Add(1)
	var fn js.Func
	fn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// args[0].Call("preventDefault") // debug in other browsers
		// args[0].Set("returnValue", "") // debug in chromium
		cancelFunc()
		fn.Release()
		wg.Done()
		return nil
	})
	global.Call("addEventListener", "beforeunload", fn)
}

// enableInteraction removes the disabled attribute from all submit buttons, allowing users to sign in and send other forms.
func enableInteraction(dom ui.DOM) {
	document := dom.QuerySelector("body")
	submitButtons := dom.QuerySelectorAll(document, `input[type="submit"]`)
	for _, submitButton := range submitButtons {
		submitButton.Set("disabled", false)
	}
}

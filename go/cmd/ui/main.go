// +build js

package main

import (
	"context"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/go/ui"
)

func main() {
	ctx := context.Background()
	ctx, cancelFunc := context.WithCancel(ctx)
	var wg sync.WaitGroup
	beforeUnload(cancelFunc, &wg)
	ui.Init(ctx, &wg)
	wg.Wait()
}

func beforeUnload(cancelFunc context.CancelFunc, wg *sync.WaitGroup) {
	var fn js.Func
	fn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// args[0].Call("preventDefault") // debug in other browsers
		// args[0].Set("returnValue", "") // debug in chrome
		cancelFunc()
		fn.Release()
		return nil
	})
	js.Global().Call("addEventListener", "beforeunload", fn)
}

// +build js

package main

import (
	"context"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/go/ui"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/log"
)

func main() {
	ctx := context.Background()
	ctx, cancelFunc := context.WithCancel(ctx)
	var wg sync.WaitGroup
	beforeUnload(cancelFunc, &wg)
	log.InitDom(ctx, &wg)
	ui.InitDom(ctx, &wg) // TODO: refactor out  (maybe make initDom() function in this file that initializes everything)
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

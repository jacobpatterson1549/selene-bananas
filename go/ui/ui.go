// +build js

// Package ui contains js initialization logic.
package ui

import (
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/go/ui/content"
)

// Init initializes the ui by regestering js functions.
func Init() {
	global := js.Global()
	funcs := make(map[string]js.Func)
	addFunc := func(parentName, fnName string, fn func(this js.Value, args []js.Value) interface{}) {
		parent := global.Get(parentName)
		jsFunc := js.FuncOf(fn)
		parent.Set(fnName, jsFunc)
		key := parentName + "." + fnName
		// if _, ok := funcs[key]; ok {
		// 	panic("duplicate function name: " + key)
		// }
		funcs[key] = jsFunc
	}

	// content
	global.Set("content", js.ValueOf(make(map[string]interface{})))
	addFunc("content", "setLoggedIn", func(this js.Value, args []js.Value) interface{} {
		loggedIn := args[0].Bool()
		content.SetLoggedIn(loggedIn)
		return nil
	})
	addFunc("content", "isLoggedIn", func(this js.Value, args []js.Value) interface{} {
		loggedIn := content.IsLoggedIn()
		return js.ValueOf(loggedIn)
	})
	addFunc("content", "setErrorMessage", func(this js.Value, args []js.Value) interface{} {
		text := args[0].String()
		content.SetErrorMessage(text)
		return nil
	})
	addFunc("content", "getJWT", func(this js.Value, args []js.Value) interface{} {
		jwt := content.GetJWT()
		return js.ValueOf(jwt)
	})
	addFunc("content", "setJWT", func(this js.Value, args []js.Value) interface{} {
		jwt := args[0].String()
		content.SetJWT(jwt)
		return nil
	})
	// log
	// lobby
	// user
	// websocket
	// canvas
	// game

	var onClose js.Func
	onClose = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		for _, fn := range funcs {
			fn.Release()
		}
		onClose.Release()
		return nil
	})
	global.Set("onclose", onClose)

}

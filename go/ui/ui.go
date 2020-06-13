// +build js

// Package ui contains js initialization logic.
package ui

import (
	"context"
	"encoding/json"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/go/game"
	"github.com/jacobpatterson1549/selene-bananas/go/game/board"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/canvas"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/controller"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/log"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/socket"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/user"
)

// InitDom initializes the ui by registering js functions.
func InitDom(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	global := js.Global()
	funcs := make(map[string]js.Func)
	addFunc := func(parentName, fnName string, fn func(this js.Value, args []js.Value) interface{}) {
		parent := global.Get(parentName)
		jsFunc := js.FuncOf(fn)
		parent.Set(fnName, jsFunc)
		key := parentName + "." + fnName
		if _, ok := funcs[key]; ok {
			panic("duplicate function name: " + key)
		}
		funcs[key] = jsFunc
	}
	addWebsocketFunc := func(_websocket js.Value, fnName string, fn func(this js.Value, args []js.Value) interface{}) {
		jsFunc := js.FuncOf(fn)
		_websocket.Set(fnName, jsFunc)
		key := "websocket._websocket." + fnName
		if _, ok := funcs[key]; ok {
			panic("duplicate function name: " + key)
		}
		funcs[key] = jsFunc
	}
	// user
	global.Set("user", js.ValueOf(make(map[string]interface{})))
	addFunc("user", "logout", func(this js.Value, args []js.Value) interface{} {
		user.Logout()
		return nil
	})
	addFunc("user", "request", func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		event.Call("preventDefault")
		form := event.Get("target")
		method := form.Get("method").String()
		url := form.Get("action").String()
		origin := global.Get("location").Get("origin").String()
		urlSuffixIndex := len(origin)
		urlSuffix := url[urlSuffixIndex:]
		formInputs := form.Call("querySelectorAll", `input:not([type="submit"])`)
		params := make(map[string][]string, formInputs.Length())
		for i := 0; i < formInputs.Length(); i++ {
			formInput := formInputs.Index(i)
			name := formInput.Get("name").String()
			value := formInput.Get("value").String()
			params[name] = []string{value}
		}
		request := user.Request{
			Method:    method,
			URL:       url,
			URLSuffix: urlSuffix,
			Params:    params,
		}
		go request.Do()
		return nil
	})
	// canvas
	document := global.Get("document")
	canvasElement := document.Call("querySelector", "#game>canvas")
	contextElement := canvasElement.Call("getContext", "2d")
	canvasCtx := canvasContext{contextElement}
	var board board.Board
	canvasCfg := canvas.Config{
		Width:      canvasElement.Get("width").Int(),
		Height:     canvasElement.Get("height").Int(),
		TileLength: 20,
		FontName:   "sans-serif",
	}
	canvas := canvasCfg.New(&canvasCtx, &board)
	canvas.InitDom(ctx, wg, canvasElement)
	// game
	g := controller.NewGame(&board, &canvas)
	g.InitDom(ctx, wg)
	// websocket
	global.Set("websocket", js.ValueOf(make(map[string]interface{})))
	addFunc("websocket", "connect", func(this js.Value, args []js.Value) interface{} {
		// promise := js.ValueOf("Promise")
		promise := global.Get("Promise")
		websocket := this
		_websocket := this.Get("_websocket")
		if !_websocket.IsUndefined() && !_websocket.IsNull() {
			var resolvePromise js.Func
			resolvePromise = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				resolve := args[0]
				resolve.Invoke()
				resolvePromise.Release()
				return nil
			})
			return promise.New(resolvePromise)
		}
		event := args[0]
		form := event.Get("target")
		url := form.Get("action").String()
		if len(url) >= 4 && url[:4] == "http" {
			url = "ws" + url[4:]
		}
		jwt := user.JWT()
		url = url + "?access_token=" + jwt
		var newWebsocketPromiseFunc js.Func

		newWebsocketPromiseFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resolve := args[0]
			reject := args[1]
			jsWebsocket := global.Get("WebSocket").New(url)
			websocket.Set("_websocket", jsWebsocket)
			addWebsocketFunc(jsWebsocket, "onmessage", func(this js.Value, args []js.Value) interface{} {
				event := args[0]
				jsMessage := event.Get("data")
				messageJSON := jsMessage.String()
				var message game.Message
				err := json.Unmarshal([]byte(messageJSON), &message)
				if err != nil {
					log.Error("unmarshalling message: " + err.Error())
					return nil
				}
				socket.OnMessage(message, g) // TODO: create socket struct with pointer to game
				return nil
			})
			addWebsocketFunc(jsWebsocket, "onopen", func(this js.Value, args []js.Value) interface{} {
				socket.OnOpen()
				resolve.Invoke()
				return nil
			})
			addWebsocketFunc(jsWebsocket, "onclose", func(this js.Value, args []js.Value) interface{} {
				event := args[0]
				reason := event.Get("reason")
				switch {
				case reason.IsUndefined():
					log.Error("lobby shut down")
				default:
					log.Warning("left lobby: " + reason.String())
				}
				socket.OnClose()
				return nil
			})
			addWebsocketFunc(jsWebsocket, "onerror", func(this js.Value, args []js.Value) interface{} {
				socket.OnError()
				reject.Invoke("websocket error - check browser console")
				return nil
			})
			newWebsocketPromiseFunc.Release()
			return nil
		})
		return promise.New(newWebsocketPromiseFunc)
	})
	// TODO: rename to disconnect
	addFunc("websocket", "close", func(this js.Value, args []js.Value) interface{} {
		global.Get("websocket").Call("_close")
		return nil
	})
	addFunc("websocket", "_close", func(this js.Value, args []js.Value) interface{} {
		_websocket := this.Get("_websocket")
		if _websocket.IsUndefined() || _websocket.IsNull() {
			return nil
		}
		socket.OnClose()
		eventListeners := []string{"onmessage", "onopen", "onclose", "onerror"}
		for _, n := range eventListeners {
			_websocket.Set(n, nil) // TODO: delete with 1.14
			key := "websocket._websocket." + n
			funcs[key].Release()
			delete(funcs, key)
		}
		_websocket.Call("close")
		this.Set("_websocket", nil) // TODO: delete with 1.14
		return nil
	})
	addFunc("websocket", "send", func(this js.Value, args []js.Value) interface{} {
		_websocket := this.Get("_websocket")
		if !_websocket.IsUndefined() && !_websocket.IsNull() {
			if _websocket.Get("readyState").Int() == 1 {
				message := args[0]
				JSON := global.Get("JSON")
				messageJSON := JSON.Call("stringify", message).String()
				_websocket.Call("send", messageJSON)
			}
		}
		return nil
	})
	// onload
	go func() { // onload
		// user
		// TODO : use pattern matching, inlined
		confirmPasswordElements := document.Call("querySelectorAll", "label>input.password2")
		for i := 0; i < confirmPasswordElements.Length(); i++ {
			confirmPasswordElement := confirmPasswordElements.Index(i)
			validatePasswordFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				confirmPasswordLabelElement := confirmPasswordElement.Get("parentElement")
				parentFormElement := confirmPasswordLabelElement.Get("parentElement")
				passwordElement := parentFormElement.Call("querySelector", "label>input.password1")
				validity := ""
				passwordValue := passwordElement.Get("value").String()
				confirmValue := confirmPasswordElement.Get("value").String()
				if passwordValue != confirmValue {
					validity = "Please enter the same password."
				}
				confirmPasswordElement.Call("setCustomValidity", validity)
				return nil
			})
			key := "ValidatePassword" + string(i)
			funcs[key] = validatePasswordFunc
			confirmPasswordElement.Set("onchange", validatePasswordFunc)
		}
	}()
	go releaseOnDone(ctx, wg, funcs)
}

func releaseOnDone(ctx context.Context, wg *sync.WaitGroup, funcs map[string]js.Func) {
	<-ctx.Done()
	for _, fn := range funcs {
		fn.Release()
	}
	wg.Done()
}

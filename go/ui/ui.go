// +build js

// Package ui contains js initialization logic.
package ui

import (
	"encoding/json"
	"strconv"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/go/game"
	"github.com/jacobpatterson1549/selene-bananas/go/game/board"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/canvas"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/content"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/controller"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/log"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/socket"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/user"
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
	addCanvasListenerFunc := func(canvasElement js.Value, fnName string, fn func(this js.Value, args []js.Value) interface{}) {
		jsFunc := js.FuncOf(fn)
		args := []interface{}{
			fnName,
			jsFunc,
		}
		switch fnName {
		case "touchstart", "touchmove":
			args = append(args, map[string]interface{}{
				"passive": false,
			})
		}
		canvasElement.Call("addEventListener", args...)
		key := "canvasElement." + fnName
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
		return loggedIn
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
	global.Set("log", js.ValueOf(make(map[string]interface{})))
	addFunc("log", "info", func(this js.Value, args []js.Value) interface{} {
		text := args[0].String()
		log.Info(text)
		return nil
	})
	addFunc("log", "warning", func(this js.Value, args []js.Value) interface{} {
		text := args[0].String()
		log.Warning(text)
		return nil
	})
	addFunc("log", "error", func(this js.Value, args []js.Value) interface{} {
		text := args[0].String()
		log.Error(text)
		return nil
	})
	addFunc("log", "chat", func(this js.Value, args []js.Value) interface{} {
		text := args[0].String()
		log.Chat(text)
		return nil
	})
	addFunc("log", "clear", func(this js.Value, args []js.Value) interface{} {
		log.Clear()
		return nil
	})
	addFunc("log", "formatDate", func(this js.Value, args []js.Value) interface{} {
		date := args[0]
		utc := date.Call("UTC")
		unixSec := utc.Int() / 1000
		return log.FormatDate(unixSec)
	})
	// lobby
	global.Set("lobby", js.ValueOf(make(map[string]interface{})))
	addFunc("lobby", "getGameInfos", func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		event.Call("preventDefault")
		websocket := global.Get("websocket")
		promise := websocket.Call("connect", event)
		var getGameInfos, logConnectErr js.Func
		getGameInfos = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			m := js.ValueOf(map[string]interface{}{
				"type": int(game.Infos), // TODO: hack
			})
			websocket.Call("send", m)
			getGameInfos.Release()
			logConnectErr.Release()
			return nil
		})
		logConnectErr = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			err := args[0]
			log := global.Get("log")
			log.Call("error", "connect error: "+err.String())
			getGameInfos.Release()
			logConnectErr.Release()
			return nil
		})
		promise = promise.Call("then", getGameInfos)
		promise = promise.Call("catch", logConnectErr)
		return nil
	})
	addFunc("lobby", "leave", func(this js.Value, args []js.Value) interface{} {
		websocket := global.Get("websocket")
		websocket.Call("close")
		game := global.Get("game")
		game.Call("leave")
		return nil
	})
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
	global.Set("canvas", js.ValueOf(make(map[string]interface{})))
	document := global.Get("document")
	canvasElement := document.Call("querySelector", "#game>canvas")
	ctx := canvasElement.Call("getContext", "2d")
	canvasCtx := canvasContext{ctx}
	var board board.Board
	canvasCfg := canvas.Config{
		Width:      canvasElement.Get("width").Int(),
		Height:     canvasElement.Get("height").Int(),
		TileLength: 20,
		FontName:   "sans-serif",
	}
	canvas := canvasCfg.New(&canvasCtx, &board)
	addFunc("canvas", "redraw", func(this js.Value, args []js.Value) interface{} {
		canvas.Redraw()
		return nil
	})
	addCanvasListenerFunc(canvasElement, "mousedown", func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		x := event.Get("offsetX").Int()
		y := event.Get("offsetY").Int()
		canvas.MoveStart(x, y)
		return nil
	})
	addCanvasListenerFunc(canvasElement, "mouseup", func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		x := event.Get("offsetX").Int()
		y := event.Get("offsetY").Int()
		canvas.MoveEnd(x, y)
		return nil
	})
	addCanvasListenerFunc(canvasElement, "mousemove", func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		x := event.Get("offsetX").Int()
		y := event.Get("offsetY").Int()
		canvas.MoveCursor(x, y)
		return nil
	})
	var tl touchLoc
	addCanvasListenerFunc(canvasElement, "touchstart", func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		tl.update(event)
		canvas.MoveStart(tl.x, tl.y)
		return nil
	})
	addCanvasListenerFunc(canvasElement, "touchend", func(this js.Value, args []js.Value) interface{} {
		// the event has no touches, use previous touchLoc
		canvas.MoveEnd(tl.x, tl.y)
		return nil
	})
	addCanvasListenerFunc(canvasElement, "touchmove", func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		tl.update(event)
		canvas.MoveCursor(tl.x, tl.y)
		return nil
	})
	// game
	global.Set("game", js.ValueOf(make(map[string]interface{})))
	g := controller.NewGame(&board, &canvas)
	addFunc("game", "create", func(this js.Value, args []js.Value) interface{} {
		g.Create()
		return nil
	})
	addFunc("game", "join", func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		joinGameButton := event.Get("srcElement")
		gameIdInput := joinGameButton.Get("previousElementSibling")
		idText := gameIdInput.Get("value").String()
		id, err := strconv.Atoi(idText)
		if err != nil {
			log.Error("could not get Id of game: " + err.Error())
			return nil
		}
		g.Join(id)
		return nil
	})
	addFunc("game", "leave", func(this js.Value, args []js.Value) interface{} {
		g.Leave()
		return nil
	})
	addFunc("game", "delete", func(this js.Value, args []js.Value) interface{} {
		g.Delete()
		return nil
	})
	addFunc("game", "start", func(this js.Value, args []js.Value) interface{} {
		g.Start()
		return nil
	})
	addFunc("game", "finish", func(this js.Value, args []js.Value) interface{} {
		g.Finish()
		return nil
	})
	addFunc("game", "snagTile", func(this js.Value, args []js.Value) interface{} {
		g.SnagTile()
		return nil
	})
	addFunc("game", "swapTile", func(this js.Value, args []js.Value) interface{} {
		canvas.StartSwap()
		return nil
	})
	addFunc("game", "sendChat", func(this js.Value, args []js.Value) interface{} {
		// TODO: get element from event (args[0])
		event := args[0]
		event.Call("preventDefault")
		gameChatElement := global.Get("document").Call("querySelector", "input#game-chat")
		message := gameChatElement.Get("value").String()
		gameChatElement.Set("value", "")
		g.SendChat(message)
		return nil
	})
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
		jwt := content.GetJWT()
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
					log := global.Get("log")
					log.Call("error", "unmarshalling message: "+err.Error())
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
	// onclose
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

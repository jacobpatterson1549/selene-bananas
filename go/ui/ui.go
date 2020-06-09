// +build js

// Package ui contains js initialization logic.
package ui

import (
	"encoding/json"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/go/game"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/content"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/lobby"
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
	addFunc("lobby", "setGameInfos", func(this js.Value, args []js.Value) interface{} {
		gameInfosJs := args[0]
		// TODO avoid unnecessary marshalling
		var gameInfos []game.Info
		if gameInfosJs != js.Undefined() {
			JSON := global.Get("JSON")
			gameInfosJson := JSON.Call("stringify", gameInfosJs).String()
			err := json.Unmarshal([]byte(gameInfosJson), &gameInfos)
			if err != nil {
				log := global.Get("log")
				log.Call("error", "unmarshalling gameInfosJsJson: "+err.Error())
				return nil
			}
		}
		lobby.SetGameInfos(gameInfos)
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
	// websocket
	global.Set("websocket", js.ValueOf(make(map[string]interface{})))
	addFunc("websocket", "connect", func(this js.Value, args []js.Value) interface{} {
		// promise := js.ValueOf("Promise")
		promise := global.Get("Promise")
		websocket := this
		_websocket := this.Get("_websocket")
		switch _websocket {
		case js.Undefined(), js.Null():
		default:
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
				socket.OnMessage(message)
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
				case reason == js.Undefined():
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
		switch _websocket {
		case js.Undefined(), js.Null():
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
		switch _websocket {
		case js.Undefined(), js.Null():
		default:
			if _websocket.Get("readyState").Int() == 1 {
				message := args[0]
				JSON := global.Get("JSON")
				messageJSON := JSON.Call("stringify", message).String()
				_websocket.Call("send", messageJSON)
			}
		}
		return nil
	})
	// canvas
	// global.Set("canvas", js.ValueOf(make(map[string]interface{})))
	// game
	// global.Set("game", js.ValueOf(make(map[string]interface{})))
	// onload
	go func() { // onload
		// user
		document := global.Get("document")
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
		// canvas
		global.Get("canvas").Call("init")
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

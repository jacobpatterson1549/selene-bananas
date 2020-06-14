// +build js

// Package ui contains js initialization logic.
package ui

import (
	"context"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/go/game/board"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/canvas"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/controller"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/lobby"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/socket"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/user"
)

// InitDom initializes the ui by registering js functions.
// TODO: simplify and move contents of this to go/cmd/ui/main.go.
func InitDom(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	global := js.Global()
	funcs := make(map[string]js.Func)
	user.InitDom(ctx, wg) // TODO: make struct
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
	s := socket.Socket{
		Game: &g,
	}
	s.InitDom(ctx, wg)
	l := lobby.Lobby{
		Game:   g,
		Socket: s,
	}
	l.InitDom(ctx, wg)
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

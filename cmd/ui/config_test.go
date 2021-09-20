//go:build js && wasm

package main

import (
	"context"
	"strings"
	"sync"
	"syscall/js"
	"testing"
)

func TestInitDom_registeredFuncs(t *testing.T) {
	var f flags
	ctx := context.Background()
	ctx, cancelFunc := context.WithCancel(ctx)
	var wg sync.WaitGroup
	globalFns := initGlobal(t)
	f.initDom(ctx, &wg)
	for parentName, fnNames := range wantRegisteredFuncs {
		parent := js.Global().Get(parentName)
		keys := js.Global().Get("Object").Call("keys", parent)
		keyCount := keys.Get("length").Int()
		switch keyCount {
		case len(fnNames):
			for _, fnName := range fnNames {
				if got := parent.Get(fnName).Type(); got != js.TypeFunction {
					t.Errorf("wanted %v.%v to be a function, got %v",
						parentName, fnName, got)
				}
			}
		default:
			t.Errorf("wanted %v registered funcs for %v, got %v",
				len(fnNames), parentName, keyCount)
		}
	}
	cancelFunc()
	wg.Wait() // ensure registered functions have been released
	for _, f := range globalFns {
		f.Release()
	}
}

var wantRegisteredFuncs = map[string][]string{
	"log": {
		"clear",
	},
	"user": {
		"logout",
		"request",
		"updateConfirmPattern",
	},
	"game": {
		"create",
		"createWithConfig",
		"join",
		"leave",
		"delete",
		"start",
		"finish",
		"snagTile",
		"swapTile",
		"sendChat",
		"resizeTiles",
		"refreshTileLength",
		"viewFinalBoard",
	},
	"lobby": {
		"connect",
		"leave",
	},
}

func initGlobal(t *testing.T) []js.Func {
	getCanvasContext := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return nil
	})
	addEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return nil
	})
	var canvasParent, canvasColor js.Value
	canvas := js.ValueOf(map[string]interface{}{
		"getContext":       getCanvasContext,
		"addEventListener": addEventListener,
	})
	computedStyle := js.ValueOf(map[string]interface{}{
		"color": "???",
	})
	getComputedStyle := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return computedStyle
	})
	querySelector := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		query := args[0].String()
		if query == ".game>.canvas" {
			return canvasParent
		}
		if query == ".game>.canvas>canvas" {
			return canvas
		}
		if strings.HasPrefix(query, "#canvas-colors>") {
			return canvasColor
		}
		t.Errorf("unexpected call to querySelector: %v", query)
		return nil
	})
	document := js.ValueOf(map[string]interface{}{
		"querySelector": querySelector,
	})
	js.Global().Set("document", document)
	js.Global().Set("getComputedStyle", getComputedStyle)
	return []js.Func{querySelector, getCanvasContext, getComputedStyle, addEventListener}
}

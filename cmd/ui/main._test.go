//go:build js && wasm

package main

import (
	"strings"
	"syscall/js"
	"testing"
)

func TestMain(t *testing.T) {
	global = js.ValueOf(map[string]any{})
	global.Set("document", global)
	global.Set("color", "for-canvas")
	beforeUnloadRegistered := false
	inputsQueried := false
	mockInput := js.ValueOf(map[string]any{"disabled": true})
	jsFuncs := map[string]js.Func{
		"querySelector": js.FuncOf(func(this js.Value, args []js.Value) any { return global }),
		"querySelectorAll": js.FuncOf(func(this js.Value, args []js.Value) any {
			query := args[0].String()
			if strings.HasPrefix(query, "input") {
				inputsQueried = true
				return []any{mockInput}
			}
			return nil
		}),
		"getContext":       js.FuncOf(func(this js.Value, args []js.Value) any { return nil }),
		"getComputedStyle": js.FuncOf(func(this js.Value, args []js.Value) any { return global }),
		"addEventListener": js.FuncOf(func(this js.Value, args []js.Value) any {
			listenerType := args[0].String()
			if listenerType == "beforeunload" {
				fn := args[1]
				go fn.Invoke()
				beforeUnloadRegistered = true
			}
			return global
		}),
	}
	for name, jsFunc := range jsFuncs {
		global.Set(name, jsFunc)
	}
	main() // should return because beforeunload is called in a goroutine
	for _, jsFunc := range jsFuncs {
		jsFunc.Release()
	}
	if !inputsQueried || mockInput.Get("disabled").Bool() {
		t.Errorf("wanted input to be enabled")
	}
	if !beforeUnloadRegistered {
		t.Errorf("wanted beforeunload to be regestered to cleanup dom when browser is closed")
	}
}

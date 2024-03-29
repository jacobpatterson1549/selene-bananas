//go:build js && wasm

package ui

import (
	"syscall/js"
	"testing"
)

func MockQuerySelector(t *testing.T, wantQuery string, wantValue js.Value, dom *DOM) js.Func {
	t.Helper()
	querySelector := MockQuery(t, wantQuery, wantValue)
	document := js.ValueOf(map[string]interface{}{
		"querySelector": querySelector,
	})
	dom.global.Set("document", document)
	return querySelector
}

func MockQuery(t *testing.T, wantQuery string, wantValue js.Value) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		gotQuery := args[0].String()
		if wantQuery != gotQuery {
			t.Errorf("wanted query to be %v, got %v", wantQuery, gotQuery)
			return nil
		}
		return wantValue
	})
}

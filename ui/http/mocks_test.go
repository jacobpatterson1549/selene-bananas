//go:build js && wasm

package http

import "syscall/js"

type mockDOM struct {
	NewXHRFunc         func() js.Value
	NewJsEventFuncFunc func(fn func(event js.Value)) js.Func
}

func (m *mockDOM) NewXHR() js.Value {
	return m.NewXHRFunc()
}

func (m *mockDOM) NewJsEventFunc(fn func(event js.Value)) js.Func {
	return m.NewJsEventFuncFunc(fn)
}

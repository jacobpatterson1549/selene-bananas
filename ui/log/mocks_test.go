//go:build js && wasm

package log

import (
	"context"
	"sync"
	"syscall/js"
)

type mockDOM struct {
	QuerySelectorFunc func(query string) js.Value
	SetCheckedFunc    func(query string, checked bool)
	FormatTimeFunc    func(utcSeconds int64) string
	CloneElementFunc  func(query string) js.Value
	NewJsFuncFunc     func(fn func()) js.Func
	RegisterFuncsFunc func(ctx context.Context, wg *sync.WaitGroup, parentName string, jsFuncs map[string]js.Func)
}

func (m mockDOM) QuerySelector(query string) js.Value {
	return m.QuerySelectorFunc(query)
}

func (m *mockDOM) SetChecked(query string, checked bool) {
	m.SetCheckedFunc(query, checked)
}

func (m mockDOM) FormatTime(utcSeconds int64) string {
	return m.FormatTimeFunc(utcSeconds)
}

func (m mockDOM) CloneElement(query string) js.Value {
	return m.CloneElementFunc(query)
}

func (m *mockDOM) NewJsFunc(fn func()) js.Func {
	return m.NewJsFuncFunc(fn)
}

func (m *mockDOM) RegisterFuncs(ctx context.Context, wg *sync.WaitGroup, parentName string, jsFuncs map[string]js.Func) {
	m.RegisterFuncsFunc(ctx, wg, parentName, jsFuncs)
}

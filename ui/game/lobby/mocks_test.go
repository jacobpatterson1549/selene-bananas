//go:build js && wasm

package lobby

import (
	"context"
	"sync"
	"syscall/js"
)

type mockSocket struct {
	connectFunc func(event js.Value) error
	closeFunc   func()
}

func (m mockSocket) Connect(event js.Value) error {
	return m.connectFunc(event)
}

func (m mockSocket) Close() {
	m.closeFunc()
}

type mockLog struct {
	errorFunc func(text string)
}

func (m mockLog) Error(text string) {
	m.errorFunc(text)
}

type mockGame struct {
	leaveFunc func()
}

func (m mockGame) Leave() {
	m.leaveFunc()
}

type mockDOM struct {
	QuerySelectorFunc       func(query string) js.Value
	FormatTimeFunc          func(utcSeconds int64) string
	CloneElementFunc        func(query string) js.Value
	RegisterFuncsFunc       func(ctx context.Context, wg *sync.WaitGroup, parentName string, jsFuncs map[string]js.Func)
	NewJsFuncFunc           func(fn func()) js.Func
	NewJsEventFuncAsyncFunc func(fn func(event js.Value), async bool) js.Func
}

func (m mockDOM) QuerySelector(query string) js.Value {
	return m.QuerySelectorFunc(query)
}

func (m mockDOM) FormatTime(utcSeconds int64) string {
	return m.FormatTimeFunc(utcSeconds)
}

func (m mockDOM) CloneElement(query string) js.Value {
	return m.CloneElementFunc(query)
}

func (m *mockDOM) RegisterFuncs(ctx context.Context, wg *sync.WaitGroup, parentName string, jsFuncs map[string]js.Func) {
	m.RegisterFuncsFunc(ctx, wg, parentName, jsFuncs)
}

func (m *mockDOM) NewJsFunc(fn func()) js.Func {
	return m.NewJsFuncFunc(fn)
}

func (m *mockDOM) NewJsEventFuncAsync(fn func(event js.Value), async bool) js.Func {
	return m.NewJsEventFuncAsyncFunc(fn, async)
}

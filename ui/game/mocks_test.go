package game

import (
	"context"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/game/message"
)

type mockDOM struct {
	QuerySelectorFunc     func(query string) js.Value
	QuerySelectorAllFunc  func(document js.Value, query string) []js.Value
	CheckedFunc           func(query string) bool
	SetCheckedFunc        func(query string, checked bool)
	ValueFunc             func(query string) string
	SetValueFunc          func(query, value string)
	SetButtonDisabledFunc func(query string, disabled bool)
	FormatTimeFunc        func(utcSeconds int64) string
	CloneElementFunc      func(query string) js.Value
	ConfirmFunc           func(message string) bool
	ColorFunc             func(element js.Value) string
	NewWebSocketFunc      func(url string) js.Value
	NewXHRFunc            func() js.Value
	Base64DecodeFunc      func(a string) []byte
	// funcs
	RegisterFuncsFunc        func(ctx context.Context, wg *sync.WaitGroup, parentName string, jsFuncs map[string]js.Func)
	NewJsFuncFunc            func(fn func()) js.Func
	NewJsEventFuncFunc       func(fn func(event js.Value)) js.Func
	NewJsEventFuncAsyncFunc  func(fn func(event js.Value), async bool) js.Func
	ReleaseJsFuncsOnDoneFunc func(ctx context.Context, wg *sync.WaitGroup, jsFuncs map[string]js.Func)
	AlertOnPanicFunc         func()
}

func (m mockDOM) QuerySelector(query string) js.Value {
	return m.QuerySelectorFunc(query)
}

func (m mockDOM) QuerySelectorAll(document js.Value, query string) []js.Value {
	return m.QuerySelectorAllFunc(document, query)
}

func (m mockDOM) Checked(query string) bool {
	return m.CheckedFunc(query)
}

func (m *mockDOM) SetChecked(query string, checked bool) {
	m.SetCheckedFunc(query, checked)
}

func (m mockDOM) Value(query string) string {
	return m.ValueFunc(query)
}

func (m *mockDOM) SetValue(query, value string) {
	m.SetValueFunc(query, value)
}

func (m *mockDOM) SetButtonDisabled(query string, disabled bool) {
	m.SetButtonDisabledFunc(query, disabled)
}

func (m mockDOM) FormatTime(utcSeconds int64) string {
	return m.FormatTimeFunc(utcSeconds)
}

func (m mockDOM) CloneElement(query string) js.Value {
	return m.CloneElementFunc(query)
}

func (m *mockDOM) Confirm(message string) bool {
	return m.ConfirmFunc(message)
}

func (m mockDOM) Color(element js.Value) string {
	return m.ColorFunc(element)
}

func (m *mockDOM) NewWebSocket(url string) js.Value {
	return m.NewWebSocketFunc(url)
}

func (m *mockDOM) NewXHR() js.Value {
	return m.NewXHRFunc()
}

func (m *mockDOM) Base64Decode(a string) []byte {
	return m.Base64DecodeFunc(a)
}

func (m *mockDOM) RegisterFuncs(ctx context.Context, wg *sync.WaitGroup, parentName string, jsFuncs map[string]js.Func) {
	m.RegisterFuncsFunc(ctx, wg, parentName, jsFuncs)
}

func (m *mockDOM) NewJsFunc(fn func()) js.Func {
	return m.NewJsFuncFunc(fn)
}

func (m *mockDOM) NewJsEventFunc(fn func(event js.Value)) js.Func {
	return m.NewJsEventFuncFunc(fn)
}

func (m *mockDOM) NewJsEventFuncAsync(fn func(event js.Value), async bool) js.Func {
	return m.NewJsEventFuncAsyncFunc(fn, async)
}

func (m *mockDOM) ReleaseJsFuncsOnDone(ctx context.Context, wg *sync.WaitGroup, jsFuncs map[string]js.Func) {
	m.ReleaseJsFuncsOnDoneFunc(ctx, wg, jsFuncs)
}

func (m *mockDOM) AlertOnPanic() {
	m.AlertOnPanicFunc()
}

type mockLog struct {
	ErrorFunc func(text string)
	InfoFunc  func(text string)
}

func (m *mockLog) Error(text string) {
	m.ErrorFunc(text)
}

func (m *mockLog) Info(text string) {
	m.InfoFunc(text)
}

type mockSocket struct {
	SendFunc func(m message.Message)
}

func (ms *mockSocket) Send(m message.Message) {
	ms.SendFunc(m)
}

//go:build js && wasm

package user

import (
	"context"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/ui/http"
)

type mockLog struct {
	ErrorFunc   func(text string)
	WarningFunc func(text string)
	ClearFunc   func()
}

func (m *mockLog) Error(text string) {
	m.ErrorFunc(text)
}

func (m *mockLog) Warning(text string) {
	m.WarningFunc(text)
}

func (m mockLog) Clear() {
	m.ClearFunc()
}

type mockDOM struct {
	QuerySelectorFunc       func(query string) js.Value
	QuerySelectorAllFunc    func(document js.Value, query string) []js.Value
	CheckedFunc             func(query string) bool
	SetCheckedFunc          func(query string, checked bool)
	ValueFunc               func(query string) string
	SetValueFunc            func(query, value string)
	ConfirmFunc             func(message string) bool
	NewXHRFunc              func() js.Value
	StoreCredentialsFunc    func(form js.Value)
	EncodeURIComponentFunc  func(str string) string
	RegisterFuncsFunc       func(ctx context.Context, wg *sync.WaitGroup, parentName string, jsFuncs map[string]js.Func)
	Base64DecodeFunc        func(a string) []byte
	NewJsEventFuncFunc      func(fn func(event js.Value)) js.Func
	NewJsEventFuncAsyncFunc func(fn func(event js.Value), async bool) js.Func
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

func (m *mockDOM) Confirm(message string) bool {
	return m.ConfirmFunc(message)
}

func (m *mockDOM) NewXHR() js.Value {
	return m.NewXHRFunc()
}

func (m *mockDOM) Base64Decode(a string) []byte {
	return m.Base64DecodeFunc(a)
}

func (m *mockDOM) StoreCredentials(form js.Value) {
	m.StoreCredentialsFunc(form)
}

func (m mockDOM) EncodeURIComponent(str string) string {
	return m.EncodeURIComponentFunc(str)
}

func (m *mockDOM) RegisterFuncs(ctx context.Context, wg *sync.WaitGroup, parentName string, jsFuncs map[string]js.Func) {
	m.RegisterFuncsFunc(ctx, wg, parentName, jsFuncs)
}

func (m *mockDOM) NewJsEventFunc(fn func(event js.Value)) js.Func {
	return m.NewJsEventFuncFunc(fn)
}

func (m *mockDOM) NewJsEventFuncAsync(fn func(event js.Value), async bool) js.Func {
	return m.NewJsEventFuncAsyncFunc(fn, async)
}

type mockSocket struct {
	CloseFunc func()
}

func (m *mockSocket) Close() {
	m.CloseFunc()
}

type mockHTTPRequester struct {
	DoFunc func(dom http.DOM, req http.Request) (*http.Response, error)
}

func (m mockHTTPRequester) Do(dom http.DOM, req http.Request) (*http.Response, error) {
	return m.DoFunc(dom, req)
}

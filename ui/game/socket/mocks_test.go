//go:build js && wasm

package socket

import (
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
)

type mockUser struct {
	JWTFunc      func() string
	UsernameFunc func() string
	LogoutFunc   func()
}

func (m mockUser) JWT() string {
	return m.JWTFunc()
}

func (m mockUser) Username() string {
	return m.UsernameFunc()
}

func (m mockUser) Logout() {
	m.LogoutFunc()
}

type mockGame struct {
	IDFunc         func() game.ID
	LeaveFunc      func()
	UpdateInfoFunc func(msg message.Message)
}

func (m mockGame) ID() game.ID {
	return m.IDFunc()
}

func (m mockGame) Leave() {
	m.LeaveFunc()
}

func (m *mockGame) UpdateInfo(msg message.Message) {
	m.UpdateInfoFunc(msg)
}

type mockLobby struct {
	SetGameInfosFunc func(gameInfos []game.Info, username string)
}

func (m mockLobby) SetGameInfos(gameInfos []game.Info, username string) {
	m.SetGameInfosFunc(gameInfos, username)
}

type mockDOM struct {
	QuerySelectorFunc    func(query string) js.Value
	QuerySelectorAllFunc func(document js.Value, query string) []js.Value
	SetCheckedFunc       func(query string, checked bool)
	NewWebSocketFunc     func(url string) js.Value
	NewJsFuncFunc        func(fn func()) js.Func
	NewJsEventFuncFunc   func(fn func(event js.Value)) js.Func
	AlertOnPanicFunc     func()
}

func (m mockDOM) QuerySelector(query string) js.Value {
	return m.QuerySelectorFunc(query)
}

func (m mockDOM) QuerySelectorAll(document js.Value, query string) []js.Value {
	return m.QuerySelectorAllFunc(document, query)
}

func (m *mockDOM) SetChecked(query string, checked bool) {
	m.SetCheckedFunc(query, checked)
}

func (m *mockDOM) NewWebSocket(url string) js.Value {
	return m.NewWebSocketFunc(url)
}

func (m *mockDOM) NewJsFunc(fn func()) js.Func {
	return m.NewJsFuncFunc(fn)
}

func (m *mockDOM) NewJsEventFunc(fn func(event js.Value)) js.Func {
	return m.NewJsEventFuncFunc(fn)
}

func (m *mockDOM) AlertOnPanic() {
	m.AlertOnPanicFunc()
}

type mockLog struct {
	InfoFunc    func(text string)
	WarningFunc func(text string)
	ErrorFunc   func(text string)
	ChatFunc    func(text string)
}

func (m *mockLog) Info(text string) {
	m.InfoFunc(text)
}

func (m *mockLog) Warning(text string) {
	m.WarningFunc(text)
}

func (m *mockLog) Error(text string) {
	m.ErrorFunc(text)
}

func (m *mockLog) Chat(text string) {
	m.ChatFunc(text)
}

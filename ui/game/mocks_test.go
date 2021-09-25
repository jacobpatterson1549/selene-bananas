package game

import (
	"context"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/game"
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
	CloneElementFunc      func(query string) js.Value
	ConfirmFunc           func(message string) bool
	ColorFunc             func(element js.Value) string
	RegisterFuncsFunc        func(ctx context.Context, wg *sync.WaitGroup, parentName string, jsFuncs map[string]js.Func)
	NewJsFuncFunc            func(fn func()) js.Func
	NewJsEventFuncFunc       func(fn func(event js.Value)) js.Func
	ReleaseJsFuncsOnDoneFunc func(ctx context.Context, wg *sync.WaitGroup, jsFuncs map[string]js.Func)
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

func (m mockDOM) CloneElement(query string) js.Value {
	return m.CloneElementFunc(query)
}

func (m *mockDOM) Confirm(message string) bool {
	return m.ConfirmFunc(message)
}

func (m mockDOM) Color(element js.Value) string {
	return m.ColorFunc(element)
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

func (m *mockDOM) ReleaseJsFuncsOnDone(ctx context.Context, wg *sync.WaitGroup, jsFuncs map[string]js.Func) {
	m.ReleaseJsFuncsOnDoneFunc(ctx, wg, jsFuncs)
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

type mockCanvas struct {
	StartSwapFunc            func()
	RedrawFunc               func()
	SetGameStatusFunc        func(s game.Status)
	TileLengthFunc           func() int
	SetTileLengthFunc        func(tileLength int)
	ParentDivOffsetWidthFunc func() int
	UpdateSizeFunc           func(width int)
	NumRowsFunc              func() int
	NumColsFunc              func() int
}

func (m *mockCanvas) StartSwap() {
	m.StartSwapFunc()
}

func (m *mockCanvas) Redraw() {
	m.RedrawFunc()
}

func (m *mockCanvas) SetGameStatus(s game.Status) {
	m.SetGameStatusFunc(s)
}

func (m mockCanvas) TileLength() int {
	return m.TileLengthFunc()
}

func (m *mockCanvas) SetTileLength(tileLength int) {
	m.SetTileLengthFunc(tileLength)
}

func (m mockCanvas) ParentDivOffsetWidth() int {
	return m.ParentDivOffsetWidthFunc()
}

func (m *mockCanvas) UpdateSize(width int) {
	m.UpdateSizeFunc(width)
}

func (m mockCanvas) NumRows() int {
	return m.NumRowsFunc()
}

func (m mockCanvas) NumCols() int {
	return m.NumColsFunc()
}

package canvas

import "github.com/jacobpatterson1549/selene-bananas/game/message"

type mockContext struct {
	SetFontFunc        func(name string)
	SetLineWidthFunc   func(width float64)
	SetFillColorFunc   func(name string)
	SetStrokeColorFunc func(name string)
	SetOpacityFunc     func(fraction string)
	FillTextFunc       func(text string, x, y int)
	ClearRectFunc      func(x, y, width, height int)
	FillRectFunc       func(x, y, width, height int)
	StrokeRectFunc     func(x, y, width, height int)
}

func (m *mockContext) SetFont(name string) {
	m.SetFontFunc(name)
}

func (m *mockContext) SetLineWidth(width float64) {
	m.SetLineWidthFunc(width)
}

func (m *mockContext) SetFillColor(name string) {
	m.SetFillColorFunc(name)
}

func (m *mockContext) SetStrokeColor(name string) {
	m.SetStrokeColorFunc(name)
}

func (m *mockContext) FillText(text string, x, y int) {
	m.FillTextFunc(text, x, y)
}

func (m *mockContext) ClearRect(x, y, width, height int) {
	m.ClearRectFunc(x, y, width, height)
}

func (m *mockContext) FillRect(x, y, width, height int) {
	m.FillRectFunc(x, y, width, height)
}

func (m *mockContext) StrokeRect(x, y, width, height int) {
	m.StrokeRectFunc(x, y, width, height)
}

type mockSocket struct {
	SendFunc func(m message.Message)
}

func (m mockSocket) Send(msg message.Message) {
	m.SendFunc(msg)
}

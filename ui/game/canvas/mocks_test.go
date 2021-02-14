package canvas

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

func (ctx *mockContext) SetFont(name string) {
	ctx.SetFontFunc(name)
}

func (ctx *mockContext) SetLineWidth(width float64) {
	ctx.SetLineWidthFunc(width)
}

func (ctx *mockContext) SetFillColor(name string) {
	ctx.SetFillColorFunc(name)
}

func (ctx *mockContext) SetStrokeColor(name string) {
	ctx.SetStrokeColorFunc(name)
}

func (ctx *mockContext) FillText(text string, x, y int) {
	ctx.FillTextFunc(text, x, y)
}

func (ctx *mockContext) ClearRect(x, y, width, height int) {
	ctx.ClearRectFunc(x, y, width, height)
}

func (ctx *mockContext) FillRect(x, y, width, height int) {
	ctx.FillRectFunc(x, y, width, height)
}

func (ctx *mockContext) StrokeRect(x, y, width, height int) {
	ctx.StrokeRectFunc(x, y, width, height)
}

// +build js

package ui

import (
	"syscall/js"
)

type (
	canvasContext struct {
		ctx js.Value
	}
)

func (cc *canvasContext) SetFont(name string) {
	cc.ctx.Set("font", name)
}

func (cc *canvasContext) SetLineWidth(width int) {
	cc.ctx.Set("lineWidth", width)
}

func (cc *canvasContext) SetFillColor(name string) {
	cc.ctx.Set("fillStyle", name)
}

func (cc *canvasContext) SetStrokeColor(name string) {
	cc.ctx.Set("strokeStyle", name)
}

func (cc *canvasContext) FillText(text string, x, y int) {
	cc.ctx.Call("fillText", text, x, y)
}

func (cc *canvasContext) ClearRect(x, y, width, height int) {
	cc.ctx.Call("clearRect", x, y, width, height)
}

func (cc *canvasContext) FillRect(x, y, width, height int) {
	cc.ctx.Call("fillRect", x, y, width, height)
}

func (cc *canvasContext) StrokeRect(x, y, width, height int) {
	cc.ctx.Call("strokeRect", x, y, width, height)
}

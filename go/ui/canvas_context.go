// +build js

package ui

import (
	"syscall/js"
)

type (
	canvasContext struct {
		ctx js.Value
	}

	touchLoc struct {
		x int
		y int
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

func (tm *touchLoc) update(event js.Value) {
	event.Call("preventDefault")
	touches := event.Get("touches")
	if touches.Length() == 0 {
		return
	}
	touch := touches.Index(0)
	canvasRect := event.Get("target").Call("getBoundingClientRect")
	tm.x = touch.Get("pageX").Int() - canvasRect.Get("left").Int()
	tm.y = touch.Get("pageY").Int() - canvasRect.Get("top").Int()
}

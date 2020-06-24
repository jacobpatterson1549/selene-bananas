// +build js,wasm

package canvas

import (
	"syscall/js"
)

type (
	jsContext struct {
		ctx js.Value
	}
)

func (c *jsContext) SetFont(name string) {
	c.ctx.Set("font", name)
}

func (c *jsContext) SetLineWidth(width int) {
	c.ctx.Set("lineWidth", width)
}

func (c *jsContext) SetFillColor(name string) {
	c.ctx.Set("fillStyle", name)
}

func (c *jsContext) SetStrokeColor(name string) {
	c.ctx.Set("strokeStyle", name)
}

func (c *jsContext) FillText(text string, x, y int) {
	c.ctx.Call("fillText", text, x, y)
}

func (c *jsContext) ClearRect(x, y, width, height int) {
	c.ctx.Call("clearRect", x, y, width, height)
}

func (c *jsContext) FillRect(x, y, width, height int) {
	c.ctx.Call("fillRect", x, y, width, height)
}

func (c *jsContext) StrokeRect(x, y, width, height int) {
	c.ctx.Call("strokeRect", x, y, width, height)
}

//go:build js && wasm

package canvas

import (
	"reflect"
	"syscall/js"
	"testing"
)

func TestContextSetFont(t *testing.T) {
	ctx := js.ValueOf(map[string]interface{}{})
	j := jsContext{&ctx}
	want := "comic sans"
	j.SetFont(want)
	got := ctx.Get("font").String()
	if want != got {
		t.Errorf("unexpected set font value: wanted %v, got %v", want, got)
	}
}

func TestContextSetLineWidth(t *testing.T) {
	ctx := js.ValueOf(map[string]interface{}{})
	j := jsContext{&ctx}
	want := float64(16)
	j.SetLineWidth(want)
	got := ctx.Get("lineWidth").Float()
	if want != got {
		t.Errorf("unexpected set line width value: wanted %v, got %v", want, got)
	}
}

func TestContextSetFillColor(t *testing.T) {
	ctx := js.ValueOf(map[string]interface{}{})
	j := jsContext{&ctx}
	want := "gray"
	j.SetFillColor(want)
	got := ctx.Get("fillStyle").String()
	if want != got {
		t.Errorf("unexpected set fill color value: wanted %v, got %v", want, got)
	}
}

func TestContextStrokeColor(t *testing.T) {
	ctx := js.ValueOf(map[string]interface{}{})
	j := jsContext{&ctx}
	want := "yellow"
	j.SetStrokeColor(want)
	got := ctx.Get("strokeStyle").String()
	if want != got {
		t.Errorf("unexpected stroke color value: wanted %v, got %v", want, got)
	}
}

func TestContextFillText(t *testing.T) {
	funcCalled := false
	want := []js.Value{
		js.ValueOf("Hello, World!"),
		js.ValueOf(5),
		js.ValueOf(10),
	}
	f := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if got := args; !reflect.DeepEqual(want, got) {
			t.Errorf("unexpected fill text args: wanted %v, got %v", want, got)
		}
		funcCalled = true
		return nil
	})
	ctx := js.ValueOf(map[string]interface{}{
		"fillText": f,
	})
	j := jsContext{&ctx}
	j.FillText("Hello, World!", 5, 10)
	if !funcCalled {
		t.Error("fillText not called")
	}
	f.Release()
}

func TestContextClearRect(t *testing.T) {
	funcCalled := false
	want := []js.Value{
		js.ValueOf(1),
		js.ValueOf(2),
		js.ValueOf(3),
		js.ValueOf(4),
	}
	f := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if got := args; !reflect.DeepEqual(want, got) {
			t.Errorf("unexpected clear rect args: wanted %v, got %v", want, got)
		}
		funcCalled = true
		return nil
	})
	ctx := js.ValueOf(map[string]interface{}{
		"clearRect": f,
	})
	j := jsContext{&ctx}
	j.ClearRect(1, 2, 3, 4)
	if !funcCalled {
		t.Error("clearRect not called")
	}
	f.Release()
}

func TestContextFillRect(t *testing.T) {
	funcCalled := false
	want := []js.Value{
		js.ValueOf(5),
		js.ValueOf(6),
		js.ValueOf(7),
		js.ValueOf(8),
	}
	f := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if got := args; !reflect.DeepEqual(want, got) {
			t.Errorf("unexpected fill rect args: wanted %v, got %v", want, got)
		}
		funcCalled = true
		return nil
	})
	ctx := js.ValueOf(map[string]interface{}{
		"fillRect": f,
	})
	j := jsContext{&ctx}
	j.FillRect(5, 6, 7, 8)
	if !funcCalled {
		t.Error("fillRect not called")
	}
	f.Release()
}

func TestContextStrokeRect(t *testing.T) {
	funcCalled := false
	want := []js.Value{
		js.ValueOf(9),
		js.ValueOf(10),
		js.ValueOf(11),
		js.ValueOf(12),
	}
	f := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if got := args; !reflect.DeepEqual(want, got) {
			t.Errorf("unexpected stroke rect args: wanted %v, got %v", want, got)
		}
		funcCalled = true
		return nil
	})
	ctx := js.ValueOf(map[string]interface{}{
		"strokeRect": f,
	})
	j := jsContext{&ctx}
	j.StrokeRect(9, 10, 11, 12)
	if !funcCalled {
		t.Error("strokeRect not called")
	}
	f.Release()
}

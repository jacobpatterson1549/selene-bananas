//go:build js && wasm

package ui

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"syscall/js"
	"testing"
)

func TestRegisterFuncs(t *testing.T) {
	tests := []struct {
		parentName string
		jsFuncs    map[string]js.Func
	}{
		{
			parentName: "testregisterfuncs",
			jsFuncs: map[string]js.Func{
				"funcA": js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }),
			},
		},
		{
			parentName: "testregisterfuncs", // same name
			jsFuncs: map[string]js.Func{
				"funcB": js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }),
				"funcC": js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }),
			},
		},
	}
	for i, test := range tests {
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		var wg sync.WaitGroup
		dom := new(DOM)
		go cancelFunc()
		dom.RegisterFuncs(ctx, &wg, test.parentName, test.jsFuncs)
		global := js.Global()
		parent := global.Get(test.parentName)
		for jsFuncName := range test.jsFuncs {
			jsFunc := parent.Get(jsFuncName)
			if !jsFunc.Truthy() || jsFunc.Type() != js.TypeFunction {
				t.Errorf("Test %v: wanted %v.%v to be a jsFunc, got %v", i, test.parentName, jsFuncName, jsFunc)
			}
		}
		wg.Wait() // should release funcs
	}
}

func TestNewJsFunc(t *testing.T) {
	invoked := false
	fn := func() {
		invoked = true
	}
	dom := new(DOM)
	jsFunc := dom.NewJsFunc(fn)
	defer jsFunc.Release()
	jsFunc.Invoke()
	if !invoked {
		t.Error("wanted function to be invoked")
	}
}

func TestNewJsEventFunc(t *testing.T) {
	defaultPrevented, invoked := false, false
	preventDefault := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defaultPrevented = true
		return nil
	})
	event := js.ValueOf(map[string]interface{}{
		"preventDefault": preventDefault,
	})
	fn := func(event js.Value) {
		invoked = true
	}
	dom := new(DOM)
	jsFunc := dom.NewJsEventFunc(fn)
	defer jsFunc.Release()
	jsFunc.Invoke(event)
	if !defaultPrevented {
		t.Error("wanted preventDefault to be called")
	}
	if !invoked {
		t.Error("wanted function to be invoked")
	}
}

func TestNewJsEventFuncAsync(t *testing.T) {
	defaultPrevented, invoked := false, false
	preventDefault := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defaultPrevented = true
		return nil
	})
	event := js.ValueOf(map[string]interface{}{
		"preventDefault": preventDefault,
	})
	fn := func(event js.Value) {
		invoked = true
	}
	dom := new(DOM)
	jsFunc := dom.NewJsEventFuncAsync(fn, true)
	defer jsFunc.Release()
	jsFunc.Invoke(event)
	if !defaultPrevented {
		t.Error("wanted preventDefault to be called")
	}
	if !invoked {
		t.Error("wanted function to be invoked")
	}
}

func TestReleaseJsFuncsOnDone(t *testing.T) {
	jsFuncs := map[string]js.Func{
		"a": {},
		"b": {},
	}
	ctx := context.Background()
	ctx, cancelFunc := context.WithCancel(ctx)
	var wg sync.WaitGroup
	wg.Add(1) // mock registration of functions
	dom := new(DOM)
	go cancelFunc()
	dom.ReleaseJsFuncsOnDone(ctx, &wg, jsFuncs)
	wg.Wait() // will block if ReleaseJsFuncsOnDone does not call Done
}

func TestAlertOnPanic(t *testing.T) {
	tests := []struct {
		name      string
		invokeFn  func()
		wantPanic bool
	}{
		{
			name: "safe",
			invokeFn: func() {
				// NOOP
			},
		},
		{
			name: "with panic",
			invokeFn: func() {
				panic("TEST: panic should cause dom alert")
			},
			wantPanic: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			panicked, alerted := false, false
			alertFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				alerted = true
				return nil
			})
			defer alertFn.Release()
			js.Global().Set("alert", alertFn)
			t.Cleanup(func() {
				if want, got := test.wantPanic, panicked; want != got {
					t.Errorf("panic states not equal: wanted %v, got %v", want, got)
				}
				if want, got := test.wantPanic, alerted; want != got {
					t.Errorf("alert should only be fired on panic: wanted %v, got %v", want, got)
				}
			})
			dom := new(DOM)
			defer func() { // AlertOnPanic should re-panic on a panic
				if r := recover(); r != nil {
					panicked = true
				}
			}()
			defer dom.AlertOnPanic()
			test.invokeFn()
		})
	}
}

func TestRecoverError(t *testing.T) {
	tests := []struct {
		r         interface{}
		want      error
		wantPanic bool
	}{
		{
			r:    errors.New("error 0"),
			want: errors.New("error 0"),
		},
		{
			r:    "error 1",
			want: errors.New("error 1"),
		},
		{
			r:         2,
			wantPanic: true,
		},
	}
	for i, test := range tests {
		t.Run("test "+strconv.Itoa(i), func(t *testing.T) {
			dom := new(DOM)
			defer func() {
				r := recover()
				switch {
				case r == nil && test.wantPanic:
					t.Errorf("wanted panic B")
				case r != nil && !test.wantPanic:
					t.Error("unwanted panic")
				}
			}()
			got := dom.recoverError(test.r)
			switch {
			case test.wantPanic:
				t.Error("wanted panic A")
			case test.want.Error() != got.Error():
				t.Errorf("errors not equal: wanted %v, got %v", test.want, got)
			}
		})
	}
}

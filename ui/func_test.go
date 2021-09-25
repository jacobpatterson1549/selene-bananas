//go:build js && wasm

package ui

import (
	"errors"
	"strconv"
	"syscall/js"
	"testing"
)

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

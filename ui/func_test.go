package ui

import (
	"errors"
	"strconv"
	"testing"
)

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

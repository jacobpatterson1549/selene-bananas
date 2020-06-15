package controller

import (
	"testing"
)

func TestGameWarningError(t *testing.T) {
	want := "x"
	err := gameWarning(want)
	got := err.Error()
	if want != got {
		t.Errorf("wanted gameWarning error string to be '%v', but was '%v'", want, got)
	}
}

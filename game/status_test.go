package game

import (
	"testing"
)

func TestStringStatus(t *testing.T) {
	statuses := []Status{
		NotStarted,
		InProgress,
		InProgress,
		InProgress,
		Finished,
		-1,
	}
	statusStrings := make(map[string]struct{})
	for _, s := range statuses {
		statusStrings[s.String()] = struct{}{}
	}
	want := 4
	got := len(statusStrings)
	if want != got {
		t.Errorf("wanted %v unique status strings, got %v", want, got)
	}
}
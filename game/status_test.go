package game

import (
	"testing"
)

func TestStatusString(t *testing.T) {
	t.Run("uniqueStatusStrings", func(t *testing.T) {
		statuses := []Status{
			NotStarted,
			InProgress,
			InProgress,
			InProgress,
			Finished,
			Deleted,
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
	})
	t.Run("unknownStatusString", func(t *testing.T) {
		unknownStatuses := []Status{
			0,
			Deleted,
		}
		want := "?"
		for i, s := range unknownStatuses {
			got := s.String()
			if want != got {
				t.Errorf("Test %v: wanted status string of '%v' for status %v, got '%v'", i, want, s, got)
			}
		}
	})
}

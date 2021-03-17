package message

import (
	"log"
	"math/rand"
	"strings"
	"testing"
)

func TestSend(t *testing.T) {
	sendTests := []struct {
		debug bool
	}{
		{},
		{debug: true},
	}
	for i, test := range sendTests {
		out := make(chan (Message), 1)
		var b strings.Builder
		log := log.New(&b, "", 0)
		info := "TestSend"
		m := Message{Info: info}
		rand.Seed(0)
		wantID := "8717895732742165505"
		Send(m, out, test.debug, log)
		switch {
		case len(out) != 1:
			t.Errorf("Test %v: wanted 1 message to be sent, got %v", i, len(out))
		case !test.debug && b.Len() != 0:
			t.Errorf("Test %v: wanted no log message when not debugging, got %v", i, b.String())
		case test.debug && strings.Count(b.String(), "\n") != 2:
			t.Errorf("Test %v: wanted 2 lines logged, got: %v", i, b.String())
		case test.debug && strings.Count(b.String(), wantID) != 2:
			t.Errorf("Test %v: wanted id %v logged twice, got: %v", i, wantID, b.String())
		case test.debug && strings.Count(b.String(), info) != 1:
			t.Errorf("Test %v: wanted message to be logged once, got %v", i, b.String())
		}
	}
}

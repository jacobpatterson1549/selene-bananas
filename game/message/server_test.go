package message

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/server/log/logtest"
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
		log := logtest.NewLogger()
		info := "TestSend"
		m := Message{Info: info}
		rand.Seed(0)
		wantID := "8717895732742165505"
		Send(m, out, test.debug, log)
		switch {
		case len(out) != 1:
			t.Errorf("Test %v: wanted 1 message to be sent, got %v", i, len(out))
		case !test.debug && !log.Empty():
			t.Errorf("Test %v: wanted no log message when not debugging, got %v", i, log.String())
		case test.debug && strings.Count(log.String(), wantID) != 2:
			t.Errorf("Test %v: wanted id %v logged twice, got: %v", i, wantID, log.String())
		case test.debug && strings.Count(log.String(), info) != 1:
			t.Errorf("Test %v: wanted message to be logged once, got %v", i, log.String())
		}
	}
}

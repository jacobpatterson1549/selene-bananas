package message

import (
	"strconv"
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
		log := new(logtest.Logger)
		info := "TestSend"
		m := Message{Info: info}
		wantID := 8717895732742165505
		wantIDText := strconv.Itoa(wantID)
		sendDebugID = func() int { return wantID }
		Send(m, out, test.debug, log)
		switch {
		case len(out) != 1:
			t.Errorf("Test %v: wanted 1 message to be sent, got %v", i, len(out))
		case !test.debug && !log.Empty():
			t.Errorf("Test %v: wanted no log message when not debugging, got %v", i, log.String())
		case test.debug && strings.Count(log.String(), wantIDText) != 2:
			t.Errorf("Test %v: wanted id %v logged twice, got: %v", i, wantIDText, log.String())
		case test.debug && strings.Count(log.String(), info) != 1:
			t.Errorf("Test %v: wanted message to be logged once, got %v", i, log.String())
		}
	}
}

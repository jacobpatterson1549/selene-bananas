package socket

import (
	"io/ioutil"
	"log"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jacobpatterson1549/selene-bananas/game"
)

func TestNewSocket(t *testing.T) {
	log := log.New(ioutil.Discard, "test", log.LstdFlags)
	conn := new(websocket.Conn)
	timeFunc := func() int64 { return 0 }
	playerName := game.PlayerName("selene")
	cfg := Config{
		Log:            log,
		TimeFunc:       timeFunc,
		ReadWait:       20 * time.Second,
		WriteWait:      10 * time.Second,
		IdlePeriod:     3 * time.Minute,
		HTTPPingPeriod: 14 * time.Minute,
	}
	s, err := cfg.NewSocket(conn, playerName)
	switch {
	case err != nil:
		t.Errorf("unexpected error: %v", err)
	case s.pingPeriod <= 0,
		s.pingPeriod >= s.readWait:
		t.Errorf("ping period should be initialized to be less than readWait (%v)", s.readWait)
	}
}

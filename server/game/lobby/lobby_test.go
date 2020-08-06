package lobby

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game"
)

func TestNew(t *testing.T) {
	newLobbyTests := []struct {
		log        *log.Logger
		maxGames   int
		maxSockets int
		wantOk     bool
	}{
		{},
		{
			log: log.New(ioutil.Discard, "test", log.LstdFlags),
		},
		{
			log:      log.New(ioutil.Discard, "test", log.LstdFlags),
			maxGames: 4,
		},
		{
			log:        log.New(ioutil.Discard, "test", log.LstdFlags),
			maxGames:   2,
			maxSockets: 16,
			wantOk:     true,
		},
	}
	for i, test := range newLobbyTests {
		cfg := Config{
			Log:        test.log,
			MaxGames:   test.maxGames,
			MaxSockets: test.maxSockets,
		}
		l, err := cfg.NewLobby()
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: expected error", i)
		case test.maxGames != l.maxGames,
			test.maxSockets != l.maxSockets,
			l.upgrader == nil:
			t.Errorf("Test %v: values not initialized properly", i)
		}
	}
}

func TestAddUser(t *testing.T) {
	var w http.ResponseWriter
	var r http.Request
	addUserTests := []struct {
		username  string
		resultErr error
	}{
		{},
		{
			username: "fred",
		},
		{
			username:  "dino",
			resultErr: errors.New("humans only"),
		},
	}
	for i, test := range addUserTests {
		l := Lobby{
			addSockets: make(chan playerSocket, 1),
		}
		go func() {
			ps := <-l.addSockets
			if string(ps.playerName) != test.username {
				t.Errorf("Test %v: wanted player name to be %v, got %v", i, test.username, ps.playerName)
			}
			ps.result <- test.resultErr
		}()
		gotErr := l.AddUser(test.username, w, &r)
		if test.resultErr != gotErr {
			t.Errorf("Test %v: wanted error '%v', got '%v'", i, test.resultErr, gotErr)
		}
	}
}

func TestRemoveUser(t *testing.T) {
	username := "barney"
	l := Lobby{
		socketMessages: make(chan game.Message, 1),
	}
	l.RemoveUser(username)
	m := <-l.socketMessages
	switch {
	case m.Type != game.PlayerDelete:
		t.Errorf("wanted player delete messageType (%v), got %v", game.PlayerDelete, m.Type)
	case string(m.PlayerName) != username:
		t.Errorf("wanted playerName %v in message, got %v", username, m.PlayerName)
	}
}

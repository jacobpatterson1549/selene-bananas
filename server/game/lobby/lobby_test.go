package lobby

import (
	"context"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
)

type mockManager struct {
	RunFunc func(ctx context.Context, in <-chan message.Message) <-chan message.Message
}

func (m *mockManager) Run(ctx context.Context, in <-chan message.Message) <-chan message.Message {
	return m.RunFunc(ctx, in)
}

func TestNewLobby(t *testing.T) {
	testLog := log.New(ioutil.Discard, "test", log.LstdFlags)
	testSocketManager := mockManager{
		RunFunc: func(ctx context.Context, in <-chan message.Message) <-chan message.Message {
			t.Error("sm run called")
			return nil
		},
	}
	testGameManeger := mockManager{
		RunFunc: func(ctx context.Context, in <-chan message.Message) <-chan message.Message {
			t.Error("gm run called")
			return nil
		},
	}
	newLobbyTests := []struct {
		wantOk bool
		want   *Lobby
		sm     SocketManager
		gm     GameManager
		Config
	}{
		{ // no log
		},
		{ // no socket manager
			Config: Config{
				Log: testLog,
			},
		},
		{ // no game manager
			Config: Config{
				Log: testLog,
			},
			sm: &testSocketManager,
		},
		{
			Config: Config{
				Log: testLog,
			},
			sm:     &testSocketManager,
			gm:     &testGameManeger,
			wantOk: true,
			want: &Lobby{
				socketManager: &testSocketManager,
				gameManager:   &testGameManeger,
				games:         map[game.ID]game.Info{},
				Config: Config{
					Log: testLog,
				},
			},
		},
	}
	for i, test := range newLobbyTests {
		got, err := test.Config.NewLobby(test.sm, test.gm)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case !reflect.DeepEqual(test.want, got):
			t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, test.want, got)
		}
	}
}

func TestRun(t *testing.T) {
	socketManagerRun := false
	gameManagerRun := false
	l := Lobby{
		socketManager: &mockManager{
			RunFunc: func(ctx context.Context, in <-chan message.Message) <-chan message.Message {
				socketManagerRun = true
				return nil
			},
		},
		gameManager: &mockManager{
			RunFunc: func(ctx context.Context, in <-chan message.Message) <-chan message.Message {
				gameManagerRun = true
				return nil
			},
		},
	}
	ctx := context.Background()
	ctx, cancelFunc := context.WithCancel(ctx)
	err := l.Run(ctx)
	switch {
	case err != nil:
		t.Errorf("unwanted error running lobby: %v", err)
	case !socketManagerRun:
		t.Errorf("wanted socket manager to be run")
	case !gameManagerRun:
		t.Errorf("wanted game manager to be run")
	}
	ctx2 := context.Background()
	err = l.Run(ctx2)
	if err == nil {
		t.Errorf("wanted error while trying to run lobby while it is currently running")
	}
	cancelFunc()
	<-l.socketMessages // wait for it to be closed
	ctx3 := context.Background()
	err = l.Run(ctx3)
	if err == nil {
		t.Errorf("wanted error while trying to run lobby while after it has run")
	}
}

func TestAddUser(t *testing.T) {
	n := "selene"
	addUserTests := []struct {
		running      bool
		pn           player.Name
		w            http.ResponseWriter
		r            *http.Request
		addSocketErr error
		wantOk       bool
		wantM        message.Message
	}{
		{},
		{
			running: true,
			wantM: message.Message{
				Type:             message.AddSocket,
				PlayerName:       player.Name(n),
				AddSocketRequest: &message.AddSocketRequest{},
			},
			addSocketErr: errors.New("add socket error"),
		},
		{
			running: true,
			wantOk:  true,
			wantM: message.Message{
				Type:             message.AddSocket,
				PlayerName:       player.Name(n),
				AddSocketRequest: &message.AddSocketRequest{},
			},
		},
	}
	for i, test := range addUserTests {
		smOut := make(chan message.Message)
		l := &Lobby{
			socketManager: &mockManager{
				RunFunc: func(ctx context.Context, in <-chan message.Message) <-chan message.Message {
					return smOut
				},
			},
			gameManager: &mockManager{
				RunFunc: func(ctx context.Context, in <-chan message.Message) <-chan message.Message {
					return nil
				},
			},
		}
		w := httptest.NewRecorder()
		r := new(http.Request)
		go func() {
			gotM := <-l.socketMessages
			if gotM.AddSocketRequest == nil {
				t.Errorf("Test %v: AddSocketRequest not set in message", i)
				return
			}
			test.wantM.AddSocketRequest.ResponseWriter = w
			test.wantM.AddSocketRequest.Request = r
			test.wantM.AddSocketRequest.Result = gotM.AddSocketRequest.Result // created when called
			if !reflect.DeepEqual(test.wantM, gotM) {
				t.Errorf("Test %v: messages not equal\nwanted: %v\ngot:    %v", i, test.wantM, gotM)
			}
			gotM.AddSocketRequest.Result <- test.addSocketErr
		}()
		if test.running {
			ctx := context.Background()
			if err := l.Run(ctx); err != nil {
				t.Errorf("Test %v: unwanted error running lobby: %v", i, err)
				continue
			}
		}
		err := l.AddUser(n, w, r)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		}
	}
}

func TestRemoveUser(t *testing.T) {
	removeUserTests := []struct {
		running    bool
		playerName string
		wantOk     bool
		wantM      message.Message
	}{
		// {},
		{
			running:    true,
			playerName: "selene",
			wantOk:     true,
			wantM: message.Message{
				Type:       message.PlayerDelete,
				PlayerName: player.Name("selene"),
			},
		},
	}
	for i, test := range removeUserTests {
		smOut := make(chan message.Message)
		l := &Lobby{
			socketManager: &mockManager{
				RunFunc: func(ctx context.Context, in <-chan message.Message) <-chan message.Message {
					return smOut
				},
			},
			gameManager: &mockManager{
				RunFunc: func(ctx context.Context, in <-chan message.Message) <-chan message.Message {
					return nil
				},
			},
		}
		go func() {
			gotM := <-l.socketMessages
			if !reflect.DeepEqual(test.wantM, gotM) {
				t.Errorf("Test %v: messages not equal\nwanted: %v\ngot:    %v", i, test.wantM, gotM)
			}
		}()
		if test.running {
			ctx := context.Background()
			if err := l.Run(ctx); err != nil {
				t.Errorf("Test %v: unwanted error running lobby: %v", i, err)
				continue
			}
		}
		err := l.RemoveUser(test.playerName)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		}
	}
}

func TestHandleSocketMessage(t *testing.T) {
	smOut := make(chan message.Message)
	var wg sync.WaitGroup
	want := message.Message{
		Info: "test message",
	}
	l := Lobby{
		socketManager: &mockManager{
			RunFunc: func(ctx context.Context, in <-chan message.Message) <-chan message.Message {
				return smOut
			},
		},
		gameManager: &mockManager{
			RunFunc: func(ctx context.Context, in <-chan message.Message) <-chan message.Message {
				go func() {
					got := <-in
					if !reflect.DeepEqual(want, got) {
						t.Errorf("messages not equal\nwanted: %v\ngot:    %v", want, got)
					}
					wg.Done()
				}()
				return nil
			},
		},
	}
	ctx := context.Background()
	err := l.Run(ctx)
	if err != nil {
		t.Errorf("unwanted error: %v", err)
	}
	wg.Add(1)
	smOut <- want
	wg.Wait()
}

func TestHandleGameMessage(t *testing.T) {
	handleGameMessageTests := []struct {
		gameM     message.Message
		wantM     message.Message
		games     map[game.ID]game.Info
		wantGames map[game.ID]game.Info
	}{
		{ // basic message
			gameM: message.Message{
				Info: "test 0",
			},
			wantM: message.Message{
				Info: "test 0",
			},
		},
		{ // no Game on message
			gameM: message.Message{
				Type: message.Infos,
			},
			wantM: message.Message{
				Type: message.SocketError,
			},
		},
		{ // single game info added
			gameM: message.Message{
				Type: message.Infos,
				Game: &game.Info{ID: 1, Status: game.NotStarted, Players: []string{"selene"}},
			},
			wantM: message.Message{
				Type: message.Infos,
				Games: []game.Info{
					{ID: 1, Status: game.NotStarted, Players: []string{"selene"}},
				},
			},
			games: make(map[game.ID]game.Info, 1),
			wantGames: map[game.ID]game.Info{
				1: {ID: 1, Status: game.NotStarted, Players: []string{"selene"}},
			},
		},
		{ // multiple game infos, test sorted
			gameM: message.Message{
				Type: message.Infos,
				Game: &game.Info{ID: 2, Status: game.Finished},
			},
			wantM: message.Message{
				Type: message.Infos,
				Games: []game.Info{
					{ID: 1, Status: game.NotStarted},
					{ID: 2, Status: game.Finished},
					{ID: 3, Status: game.InProgress},
				},
			},
			games: map[game.ID]game.Info{
				1: {ID: 1, Status: game.NotStarted},
				3: {ID: 3, Status: game.InProgress},
				2: {ID: 2, Status: game.InProgress},
			},
			wantGames: map[game.ID]game.Info{
				3: {ID: 3, Status: game.InProgress},
				2: {ID: 2, Status: game.Finished},
				1: {ID: 1, Status: game.NotStarted},
			},
		},
		{ // game deleted
			gameM: message.Message{
				Type: message.Infos,
				Game: &game.Info{ID: 1, Status: game.Deleted},
			},
			wantM: message.Message{
				Type:  message.Infos,
				Games: []game.Info{},
			},
			games: map[game.ID]game.Info{
				1: {ID: 1, Status: game.Finished},
			},
			wantGames: map[game.ID]game.Info{},
		},
	}
	for i, test := range handleGameMessageTests {
		gmOut := make(chan message.Message)
		var wg sync.WaitGroup
		l := Lobby{
			socketManager: &mockManager{
				RunFunc: func(ctx context.Context, in <-chan message.Message) <-chan message.Message {
					go func() {
						gotM := <-in
						switch test.wantM.Type {
						case message.SocketError:
							if gotM.Type != message.SocketError {
								t.Errorf("Test %v: wanted type of socket error, got %v", i, gotM.Type)
							}
						default:
							if !reflect.DeepEqual(test.wantM, gotM) {
								t.Errorf("Test %v: messages not equal\nwanted: %v\ngot:    %v", i, test.wantM, gotM)
							}
						}
						wg.Done()
					}()
					return nil
				},
			},
			gameManager: &mockManager{
				RunFunc: func(ctx context.Context, in <-chan message.Message) <-chan message.Message {
					return gmOut
				},
			},
			games: test.games,
			Config: Config{
				Log: log.New(ioutil.Discard, "test", log.LstdFlags),
			},
		}
		ctx := context.Background()
		err := l.Run(ctx)
		if err != nil {
			t.Errorf("Test %v: unwanted error: %v", i, err)
		}
		wg.Add(1)
		gmOut <- test.gameM
		wg.Wait()
		gotGames := l.games
		if !reflect.DeepEqual(test.wantGames, gotGames) {
			t.Errorf("Test %v: game infos in lobby not equal\nwanted: %v\ngot:    %v", i, test.wantGames, gotGames)
		}
	}
}

package lobby

import (
	"context"
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

type mockRunner struct {
	RunFunc func(ctx context.Context, in <-chan message.Message) <-chan message.Message
}

func (m *mockRunner) Run(ctx context.Context, in <-chan message.Message) <-chan message.Message {
	return m.RunFunc(ctx, in)
}

func TestNewLobby(t *testing.T) {
	testLog := log.New(ioutil.Discard, "test", log.LstdFlags)
	testSocketRunner := mockRunner{
		RunFunc: func(ctx context.Context, in <-chan message.Message) <-chan message.Message {
			t.Error("sm run called")
			return nil
		},
	}
	testGameManeger := mockRunner{
		RunFunc: func(ctx context.Context, in <-chan message.Message) <-chan message.Message {
			t.Error("gm run called")
			return nil
		},
	}
	newLobbyTests := []struct {
		wantOk       bool
		want         *Lobby
		socketRunner Runner
		gameRunner   Runner
		Config
	}{
		{ // no log
		},
		{ // no socket runner
			Config: Config{
				Log: testLog,
			},
		},
		{ // no game runner
			Config: Config{
				Log: testLog,
			},
			socketRunner: &testSocketRunner,
		},
		{
			Config: Config{
				Log: testLog,
			},
			socketRunner: &testSocketRunner,
			gameRunner:   &testGameManeger,
			wantOk:       true,
			want: &Lobby{
				socketRunner: &testSocketRunner,
				gameRunner:   &testGameManeger,
				games:        map[game.ID]game.Info{},
				Config: Config{
					Log: testLog,
				},
			},
		},
	}
	for i, test := range newLobbyTests {
		got, err := test.Config.NewLobby(test.socketRunner, test.gameRunner)
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
	socketRunnerRun := false
	gameRunnerRun := false
	l := Lobby{
		socketRunner: &mockRunner{
			RunFunc: func(ctx context.Context, in <-chan message.Message) <-chan message.Message {
				socketRunnerRun = true
				return nil
			},
		},
		gameRunner: &mockRunner{
			RunFunc: func(ctx context.Context, in <-chan message.Message) <-chan message.Message {
				gameRunnerRun = true
				return nil
			},
		},
	}
	ctx := context.Background()
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()
	l.Run(ctx)
	switch {
	case !socketRunnerRun:
		t.Errorf("wanted socket runner to be run")
	case !gameRunnerRun:
		t.Errorf("wanted game runner to be run")
	default:
		cancelFunc()
		<-l.socketMessages // wait for it to be closed
	}
}

func TestAddUser(t *testing.T) {
	n := "selene"
	addUserTests := []struct {
		wantOk bool
	}{
		{},
		{
			wantOk: true,
		},
	}
	for i, test := range addUserTests {
		l := &Lobby{
			socketRunner: &mockRunner{
				RunFunc: func(ctx context.Context, in <-chan message.Message) <-chan message.Message {
					return nil
				},
			},
			gameRunner: &mockRunner{
				RunFunc: func(ctx context.Context, in <-chan message.Message) <-chan message.Message {
					return nil
				},
			},
		}
		w := httptest.NewRecorder()
		r := new(http.Request)
		var wg sync.WaitGroup
		wg.Add(1)
		go func() { // mock socket runner
			defer wg.Done()
			gotM, ok := <-l.socketMessages
			if !ok || gotM.Type != message.SocketAdd || gotM.AddSocketRequest == nil {
				t.Errorf("Test %v: AddSocketRequest not set in message: %v", i, gotM)
				return
			}
			wantM := message.Message{
				Type:       message.SocketAdd,
				PlayerName: player.Name("selene"),
				AddSocketRequest: &message.AddSocketRequest{
					ResponseWriter: w,
					Request:        r,
					Result:         gotM.AddSocketRequest.Result, // created when called,
				},
			}
			if !reflect.DeepEqual(wantM, gotM) {
				t.Errorf("Test %v: messages not equal\nwanted: %v\ngot:    %v", i, wantM, gotM)
			}
			var addSocketM message.Message
			if !test.wantOk {
				addSocketM.Type = message.SocketError
				addSocketM.Info = "add socket error"
			}
			gotM.AddSocketRequest.Result <- addSocketM
			if !test.wantOk {
				return
			}
			gotM = <-l.socketMessages
			if gotM.Type != message.GameInfos {
				t.Errorf("Test %v: wanted infos message to be sent for user, got %v", i, gotM)
			}
		}()
		ctx := context.Background()
		l.Run(ctx)
		err := l.AddUser(n, w, r)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		}
		wg.Wait()
	}
}

func TestRemoveUser(t *testing.T) {
	removeUserTests := []struct {
		running    bool
		playerName string
		wantM      message.Message
	}{
		{
			playerName: "selene",
			wantM: message.Message{
				Type:       message.PlayerDelete,
				PlayerName: player.Name("selene"),
			},
		},
	}
	for i, test := range removeUserTests {
		l := &Lobby{
			socketRunner: &mockRunner{
				RunFunc: func(ctx context.Context, in <-chan message.Message) <-chan message.Message {
					return nil
				},
			},
			gameRunner: &mockRunner{
				RunFunc: func(ctx context.Context, in <-chan message.Message) <-chan message.Message {
					return nil
				},
			},
		}
		ctx := context.Background()
		l.Run(ctx)
		go l.RemoveUser(test.playerName)
		gotM := <-l.socketMessages
		if !reflect.DeepEqual(test.wantM, gotM) {
			t.Errorf("Test %v: messages not equal\nwanted: %v\ngot:    %v", i, test.wantM, gotM)
		}
	}
}

func TestHandleSocketMessage(t *testing.T) {
	smIn := make(chan message.Message)
	var wg sync.WaitGroup
	want := message.Message{
		Info: "test message",
	}
	l := Lobby{
		socketRunner: &mockRunner{
			RunFunc: func(ctx context.Context, in <-chan message.Message) <-chan message.Message {
				return smIn
			},
		},
		gameRunner: &mockRunner{
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
	l.Run(ctx)
	wg.Add(1)
	smIn <- want
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
				Type: message.GameInfos,
			},
			wantM: message.Message{
				Type: message.SocketError,
			},
		},
		{ // single game info added
			gameM: message.Message{
				Type: message.GameInfos,
				Game: &game.Info{ID: 1, Status: game.NotStarted, Players: []string{"selene"}},
			},
			wantM: message.Message{
				Type: message.GameInfos,
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
				Type: message.GameInfos,
				Game: &game.Info{ID: 2, Status: game.Finished},
			},
			wantM: message.Message{
				Type: message.GameInfos,
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
				Type: message.GameInfos,
				Game: &game.Info{ID: 1, Status: game.Deleted},
			},
			wantM: message.Message{
				Type:  message.GameInfos,
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
			socketRunner: &mockRunner{
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
			gameRunner: &mockRunner{
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
		l.Run(ctx)
		wg.Add(1)
		gmOut <- test.gameM
		wg.Wait()
		gotGames := l.games
		if !reflect.DeepEqual(test.wantGames, gotGames) {
			t.Errorf("Test %v: game infos in lobby not equal\nwanted: %v\ngot:    %v", i, test.wantGames, gotGames)
		}
	}
}

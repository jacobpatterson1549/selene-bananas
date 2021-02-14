package lobby

import (
	"context"
	"io/ioutil"
	"log"
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
		{ // ok
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
		{ // ok with debug
			Config: Config{
				Debug: true,
				Log:   testLog,
			},
			socketRunner: &testSocketRunner,
			gameRunner:   &testGameManeger,
			wantOk:       true,
			want: &Lobby{
				socketRunner: &testSocketRunner,
				gameRunner:   &testGameManeger,
				games:        map[game.ID]game.Info{},
				Config: Config{
					Debug: true,
					Log:   testLog,
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
		case got.socketRunnerIn == nil:
			t.Errorf("Test %v: socketRunnerIn channel not created", i)
		default:
			got.socketRunnerIn = nil
			if !reflect.DeepEqual(test.want, got) {
				t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, test.want, got)
			}
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
		socketRunnerIn: make(chan message.Message),
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
		<-l.socketRunnerIn // wait for it to be closed
	}
}

func TestAddUser(t *testing.T) {
	addUserTests := []struct {
		socketRunnerResult      message.Message
		wantSecondSocketMessage bool
	}{
		{
			socketRunnerResult: message.Message{
				Info: "user cerated",
			},
			wantSecondSocketMessage: true,
		},
		{
			socketRunnerResult: message.Message{
				Type: message.SocketError,
				Info: "error adding socket for user",
			},
		},
	}
	for i, test := range addUserTests {
		u := "selene"
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/addUser", nil)
		var wg sync.WaitGroup
		l := Lobby{
			socketRunner: &mockRunner{
				RunFunc: func(ctx context.Context, in <-chan message.Message) <-chan message.Message {
					go func() {
						m := <-in
						switch {
						case m.Type != message.SocketAdd, string(m.PlayerName) != u, m.AddSocketRequest == nil, m.AddSocketRequest.ResponseWriter != w, m.AddSocketRequest.Request != r, m.AddSocketRequest.Result == nil:
							t.Errorf("Test %v: wanted socket add message sent to socketRunner, got; %v", i, m)
						}
						m.AddSocketRequest.Result <- test.socketRunnerResult
						if test.socketRunnerResult.Type != message.SocketError {
							m2 := <-in
							switch {
							case m2.Type != message.GameInfos, m2.Games == nil, m2.Info != test.socketRunnerResult.Info:
								t.Errorf("Test %v: wanted copied message with infos sent back to socket runner, got %v", i, m2)
							}
							wg.Done()
						}
					}()
					return nil
				},
			},
			gameRunner: &mockRunner{
				RunFunc: func(ctx context.Context, in <-chan message.Message) <-chan message.Message {
					return nil
				},
			},
			socketRunnerIn: make(chan message.Message),
		}
		ctx := context.Background()
		l.Run(ctx)
		wg.Add(1)
		err := l.AddUser(u, w, r)
		switch {
		case err == nil:
			wg.Wait() // ensure the socketRunner recieves the desired message
		case test.socketRunnerResult.Type != message.SocketError:
			t.Errorf("Test %v: unwanted error adding user: %v", i, err)
		case test.socketRunnerResult.Info != err.Error():
			t.Errorf("Test %v: wanted error message to be '%v', got '%v'", i, test.socketRunnerResult.Info, err.Error())
		}
	}
}

func TestRemoveUser(t *testing.T) {
	username := "selene"
	want := message.Message{
		Type:       message.PlayerRemove,
		PlayerName: player.Name(username),
	}
	var wg sync.WaitGroup
	l := &Lobby{
		socketRunner: &mockRunner{
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
		gameRunner: &mockRunner{
			RunFunc: func(ctx context.Context, in <-chan message.Message) <-chan message.Message {
				return nil
			},
		},
		socketRunnerIn: make(chan message.Message),
	}
	ctx := context.Background()
	l.Run(ctx)
	wg.Add(1)
	l.RemoveUser(username)
	wg.Wait() // ensure socket runner handles remove request
}

func TestHandleSocketMessage(t *testing.T) {
	socketRunnerOut := make(chan message.Message)
	var wg sync.WaitGroup
	want := message.Message{
		Info: "test message",
	}
	l := Lobby{
		socketRunner: &mockRunner{
			RunFunc: func(ctx context.Context, in <-chan message.Message) <-chan message.Message {
				return socketRunnerOut
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
	socketRunnerOut <- want
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
			socketRunnerIn: make(chan message.Message),
			games:          test.games,
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

package lobby

import (
	"context"
	"io"
	"log"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
)

func TestNewLobby(t *testing.T) {
	testLog := log.New(io.Discard, "", 0)
	testSocketRunner := mockRunner{
		RunFunc: func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message) <-chan message.Message {
			t.Error("sm run called")
			return nil
		},
	}
	testGameManeger := mockRunner{
		RunFunc: func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message) <-chan message.Message {
			t.Error("gm run called")
			return nil
		},
	}
	newLobbyTests := []struct {
		log          *log.Logger
		wantOk       bool
		want         *Lobby
		socketRunner Runner
		gameRunner   Runner
		Config
	}{
		{ // no log
		},
		{ // no socket runner
			log: testLog,
		},
		{ // no game runner
			log:          testLog,
			socketRunner: &testSocketRunner,
		},
		{ // ok
			log:          testLog,
			socketRunner: &testSocketRunner,
			gameRunner:   &testGameManeger,
			wantOk:       true,
			want: &Lobby{
				log:          testLog,
				socketRunner: &testSocketRunner,
				gameRunner:   &testGameManeger,
				games:        map[game.ID]game.Info{},
			},
		},
		{ // ok with debug
			log: testLog,
			Config: Config{
				Debug: true,
			},
			socketRunner: &testSocketRunner,
			gameRunner:   &testGameManeger,
			wantOk:       true,
			want: &Lobby{
				log:          testLog,
				socketRunner: &testSocketRunner,
				gameRunner:   &testGameManeger,
				games:        map[game.ID]game.Info{},
				Config: Config{
					Debug: true,
				},
			},
		},
	}
	for i, test := range newLobbyTests {
		got, err := test.Config.NewLobby(test.log, test.socketRunner, test.gameRunner)
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
	runTests := []struct {
		stopFunc func(cancelFunc context.CancelFunc, socketRunnerOut, gameRunnerOut chan message.Message)
	}{
		{
			stopFunc: func(cancelFunc context.CancelFunc, socketRunnerOut, gameRunnerOut chan message.Message) {
				cancelFunc()
			},
		},
		{
			stopFunc: func(cancelFunc context.CancelFunc, socketRunnerOut, gameRunnerOut chan message.Message) {
				close(socketRunnerOut)
			},
		},
		{
			stopFunc: func(cancelFunc context.CancelFunc, socketRunnerOut, gameRunnerOut chan message.Message) {
				close(gameRunnerOut)
			},
		},
	}
	for i, test := range runTests {
		socketRunnerOut := make(chan message.Message)
		gameRunnerOut := make(chan message.Message)
		socketRunnerRun := false
		gameRunnerRun := false
		l := Lobby{
			log: log.New(io.Discard, "", 0),
			socketRunner: &mockRunner{
				RunFunc: func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message) <-chan message.Message {
					socketRunnerRun = true
					return socketRunnerOut
				},
			},
			gameRunner: &mockRunner{
				RunFunc: func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message) <-chan message.Message {
					gameRunnerRun = true
					return gameRunnerOut
				},
			},
			socketRunnerIn: make(chan message.Message),
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		defer cancelFunc()
		var wg sync.WaitGroup
		l.Run(ctx, &wg)
		switch {
		case !socketRunnerRun:
			t.Errorf("Test %v wanted socket runner to be run", i)
		case !gameRunnerRun:
			t.Errorf("Test %v: wanted game runner to be run", i)
		default:
			test.stopFunc(cancelFunc, socketRunnerOut, gameRunnerOut)
			<-l.socketRunnerIn // wait for it to be closed
		}
		wg.Wait()
	}
}

func TestAddUser(t *testing.T) {
	addUserTests := []struct {
		socketRunnerResult      message.Message
		wantSecondSocketMessage bool
		wantErr                 bool
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
			wantErr: true,
		},
	}
	for i, test := range addUserTests {
		u := "selene"
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/addUser", nil)
		socketRunnerIn := make(chan message.Message, 1)
		l := Lobby{
			log: log.New(io.Discard, "", 0),
			socketRunner: &mockRunner{
				RunFunc: func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message) <-chan message.Message {
					wg.Add(1)
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
						}
						wg.Done()
					}()
					return nil
				},
			},
			gameRunner: &mockRunner{
				RunFunc: func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message) <-chan message.Message {
					return nil
				},
			},
			socketRunnerIn: socketRunnerIn,
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		var wg sync.WaitGroup
		l.Run(ctx, &wg)
		err := l.AddUser(u, w, r)
		switch {
		case test.wantErr:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: wanted error", i)
			// other testing done above in mock socket runner
		}
		cancelFunc()
		wg.Wait()
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
		log: log.New(io.Discard, "", 0),
		socketRunner: &mockRunner{
			RunFunc: func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message) <-chan message.Message {
				go func() {
					wg.Add(1)
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
			RunFunc: func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message) <-chan message.Message {
				return nil
			},
		},
		socketRunnerIn: make(chan message.Message),
	}
	ctx := context.Background()
	ctx, cancelFunc := context.WithCancel(ctx)
	l.Run(ctx, &wg)
	l.RemoveUser(username)
	cancelFunc()
	wg.Wait() // ensure socket runner handles remove request
}

func TestHandleSocketMessage(t *testing.T) {
	socketRunnerOut := make(chan message.Message)
	var wg sync.WaitGroup
	want := message.Message{
		Info: "test message",
	}
	l := Lobby{
		log: log.New(io.Discard, "", 0),
		socketRunner: &mockRunner{
			RunFunc: func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message) <-chan message.Message {
				return socketRunnerOut
			},
		},
		gameRunner: &mockRunner{
			RunFunc: func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message) <-chan message.Message {
				wg.Add(1)
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
		socketRunnerIn: make(chan message.Message),
	}
	ctx := context.Background()
	ctx, cancelFunc := context.WithCancel(ctx)
	l.Run(ctx, &wg)
	socketRunnerOut <- want
	cancelFunc()
	wg.Wait() // let game runner handle message
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
			log: log.New(io.Discard, "", 0),
			socketRunner: &mockRunner{
				RunFunc: func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message) <-chan message.Message {
					wg.Add(1)
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
				RunFunc: func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message) <-chan message.Message {
					return gmOut
				},
			},
			socketRunnerIn: make(chan message.Message),
			games:          test.games,
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		l.Run(ctx, &wg)
		gmOut <- test.gameM
		cancelFunc()
		wg.Wait()
		gotGames := l.games
		if !reflect.DeepEqual(test.wantGames, gotGames) {
			t.Errorf("Test %v: game infos in lobby not equal\nwanted: %v\ngot:    %v", i, test.wantGames, gotGames)
		}
	}
}

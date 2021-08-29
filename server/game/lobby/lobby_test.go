package lobby

import (
	"context"
	"fmt"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
	"github.com/jacobpatterson1549/selene-bananas/server/log"
	"github.com/jacobpatterson1549/selene-bananas/server/log/logtest"
)

func TestNewLobby(t *testing.T) {
	testLog := logtest.DiscardLogger
	testSocketRunner := mockSocketRunner(func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message, inSM <-chan message.Socket) <-chan message.Message {
		t.Error("socket runner run called")
		return nil
	})
	testGameManeger := mockGameRunner(func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message) <-chan message.Message {
		t.Error("game runner run called")
		return nil
	})
	newLobbyTests := []struct {
		log          log.Logger
		wantOk       bool
		want         *Lobby
		socketRunner SocketRunner
		gameRunner   GameRunner
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
		case got.socketMessages == nil:
			t.Errorf("Test %v: socketModifyRequests channel not created", i)
		default:
			got.socketMessages = nil
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
			log: logtest.DiscardLogger,
			socketRunner: mockSocketRunner(func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message, inSM <-chan message.Socket) <-chan message.Message {
				socketRunnerRun = true
				return socketRunnerOut
			}),
			gameRunner: mockGameRunner(func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message) <-chan message.Message {
				gameRunnerRun = true
				return gameRunnerOut
			}),
			socketMessages: make(chan message.Socket),
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
		}
		wg.Wait()
	}
}

func TestAddUser(t *testing.T) {
	addUserTests := []struct {
		addSocketErr error
		wantOk       bool
	}{
		{
			addSocketErr: fmt.Errorf("error adding socket for user"),
		},
		{
			wantOk: true,
		},
	}
	for i, test := range addUserTests {
		u := "selene"
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/addUser", nil)
		handleSocketMessage := func(wg *sync.WaitGroup, inSM <-chan message.Socket) {
			sm := <-inSM
			switch {
			case sm.Type != message.SocketAdd,
				string(sm.PlayerName) != u,
				sm.ResponseWriter != w,
				sm.Request != r,
				sm.Result == nil:
				t.Errorf("Test %v: wanted socket add message sent to socketRunner, got: %v", i, sm)
			}
			sm.Result <- test.addSocketErr
			wg.Done()
		}
		l := Lobby{
			log: logtest.DiscardLogger,
			socketRunner: mockSocketRunner(func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message, inSM <-chan message.Socket) <-chan message.Message {
				wg.Add(1)
				go handleSocketMessage(wg, inSM)
				return nil
			}),
			gameRunner: mockGameRunner(func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message) <-chan message.Message {
				return nil
			}),
			socketMessages: make(chan message.Socket, 1),
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		var wg sync.WaitGroup
		l.Run(ctx, &wg)
		err := l.AddUser(u, w, r)
		switch {
		case !test.wantOk:
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
	want := message.Socket{
		Type:       message.PlayerRemove,
		PlayerName: player.Name(username),
	}
	handleModifyRequest := func(wg *sync.WaitGroup, inSM <-chan message.Socket) {
		got := <-inSM
		if !reflect.DeepEqual(want, got) {
			t.Errorf("messages not equal\nwanted: %v\ngot:    %v", want, got)
		}
		wg.Done()
	}
	l := &Lobby{
		log: logtest.DiscardLogger,
		socketRunner: mockSocketRunner(func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message, inSM <-chan message.Socket) <-chan message.Message {
			wg.Add(1)
			go handleModifyRequest(wg, inSM)
			return nil
		}),
		gameRunner: mockGameRunner(
			func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message) <-chan message.Message {
				return nil
			}),
		socketMessages: make(chan message.Socket),
	}
	ctx := context.Background()
	ctx, cancelFunc := context.WithCancel(ctx)
	var wg sync.WaitGroup
	l.Run(ctx, &wg)
	l.RemoveUser(username)
	cancelFunc()
	wg.Wait() // ensure socket runner handles remove request
}

func TestHandleSocketMessage(t *testing.T) {
	handleSocketMessageTests := []struct {
		message.Message
		wantGameM   message.Message
		wantSocketM message.Message
	}{
		{
			Message: message.Message{
				Info: "test message",
			},
			wantGameM: message.Message{
				Info: "test message",
			},
		},
		{
			Message: message.Message{
				Type: message.GameInfos,
				Addr: "test.two",
			},
			wantSocketM: message.Message{
				Type:  message.GameInfos,
				Addr:  "test.two",
				Games: []game.Info{{}, {}},
			},
		},
	}
	for i, test := range handleSocketMessageTests {
		socketRunnerOut := make(chan message.Message)
		var wg sync.WaitGroup
		var gotGameM, gotSocketM message.Message
		l := Lobby{
			log: logtest.DiscardLogger,
			socketRunner: mockSocketRunner(func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message, inSM <-chan message.Socket) <-chan message.Message {
				wg.Add(1)
				handleSocketMessage := func() {
					gotSocketM = <-in
					wg.Done()
				}
				go handleSocketMessage()
				return socketRunnerOut
			}),
			gameRunner: mockGameRunner(func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message) <-chan message.Message {
				wg.Add(1)
				handleGameMessage := func() {
					gotGameM = <-in
					wg.Done()
				}
				go handleGameMessage()
				return nil
			}),
			games: map[game.ID]game.Info{
				1: {},
				2: {},
			},
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		l.Run(ctx, &wg)
		socketRunnerOut <- test.Message
		cancelFunc()
		wg.Wait()
		switch {
		case !reflect.DeepEqual(test.wantGameM, gotGameM):
			t.Errorf("Test %v: game messages not equal\nwanted: %v\ngot:    %v", i, test.wantGameM, gotGameM)
		case !reflect.DeepEqual(test.wantSocketM, gotSocketM):
			t.Errorf("Test %v: socket messages not equal\nwanted: %v\ngot:    %v", i, test.wantSocketM, gotSocketM)
		}
	}
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
		handleGameMessages := func(wg *sync.WaitGroup, in <-chan message.Message) {
			gotM := <-in
			switch {
			case test.wantM.Type == message.SocketError:
				if gotM.Type != message.SocketError {
					t.Errorf("Test %v: wanted type of socket error, got %v", i, gotM.Type)
				}
			case !reflect.DeepEqual(test.wantM, gotM):
				t.Errorf("Test %v: messages not equal\nwanted: %v\ngot:    %v", i, test.wantM, gotM)
			}
			wg.Done()
		}
		l := Lobby{
			log: logtest.DiscardLogger,
			socketRunner: mockSocketRunner(func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message, inSM <-chan message.Socket) <-chan message.Message {
				wg.Add(1)
				go handleGameMessages(wg, in)
				return nil
			}),
			gameRunner: mockGameRunner(func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message) <-chan message.Message {
				return gmOut
			}),
			games: test.games,
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		var wg sync.WaitGroup
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

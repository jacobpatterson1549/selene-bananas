package lobby

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/jacobpatterson1549/selene-bananas/go/db"
	"github.com/jacobpatterson1549/selene-bananas/go/game"
	"github.com/jacobpatterson1549/selene-bananas/go/game/controller"
	"github.com/jacobpatterson1549/selene-bananas/go/game/socket"
	"github.com/jacobpatterson1549/selene-bananas/go/game/tile"
)

type (
	// Lobby is the place users can create, join, and participate in games
	Lobby struct {
		debug         bool
		log           *log.Logger
		upgrader      *websocket.Upgrader
		rand          *rand.Rand
		words         map[string]struct{}
		socketCfg     socket.Config
		gameCfg       controller.Config
		maxGames      int
		maxSockets    int
		addPlayers    chan playerSocket
		removePlayers chan game.PlayerName
	}

	// Config contiains the properties to create a lobby
	Config struct {
		Debug   bool
		Log     *log.Logger
		Rand    *rand.Rand
		UserDao db.UserDao
	}

	playerSocket struct {
		game.PlayerName
		socket.Socket
	}

	messageHandler struct {
		done          chan<- struct{}
		writeMessages chan<- game.Message
		readMessages  <-chan game.Message
		readErrs      <-chan error
	}
)

// NewLobby creates a new game lobby
func (cfg Config) NewLobby(ws game.WordsSupplier) (Lobby, error) {
	u := new(websocket.Upgrader)
	u.Error = func(w http.ResponseWriter, r *http.Request, status int, reason error) {
		log.Println(reason)
	}
	l := Lobby{
		debug:      cfg.Debug,
		log:        cfg.Log,
		upgrader:   u,
		rand:       cfg.Rand,
		maxGames:   4,
		maxSockets: 32,
	}
	l.socketCfg = socket.Config{
		Debug: cfg.Debug,
		Log:   l.log,
	}
	words := ws.Words()
	l.gameCfg = controller.Config{
		Debug:       cfg.Debug,
		Log:         l.log,
		UserDao:     cfg.UserDao,
		Words:       words,
		MaxPlayers:  8,
		NumNewTiles: 21,
		ShuffleUnusedTilesFunc: func(tiles []tile.Tile) {
			l.rand.Shuffle(len(tiles), func(i, j int) {
				tiles[i], tiles[j] = tiles[j], tiles[i]
			})
		},
		ShufflePlayersFunc: func(sockets []game.PlayerName) {
			l.rand.Shuffle(len(sockets), func(i, j int) {
				sockets[i], sockets[j] = sockets[j], sockets[i]
			})
		},
	}
	return l, nil
}

// AddUser adds a user to the lobby, it opens a new websocket (player) for the username
func (l *Lobby) AddUser(playerName game.PlayerName, w http.ResponseWriter, r *http.Request) error {
	conn, err := l.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return fmt.Errorf("upgrading to websocket connection: %w", err)
	}
	s := l.socketCfg.NewSocket(conn, playerName)
	ps := playerSocket{
		PlayerName: playerName,
		Socket:     s,
	}
	l.addPlayers <- ps
	return nil
}

// RemoveUser removes the user from the lobby and a game, if any
func (l *Lobby) RemoveUser(playerName game.PlayerName) {
	l.removePlayers <- playerName
}

// Run runs the lobby
func (l *Lobby) Run(done <-chan struct{}) {
	l.addPlayers = make(chan playerSocket)
	l.removePlayers = make(chan game.PlayerName)
	sockets := make(map[game.PlayerName]messageHandler, l.maxSockets)
	games := make(map[game.ID]messageHandler, l.maxGames)
	socketMessages := make(<-chan game.Message)
	gameMessages := make(<-chan game.Message)
	errs := make(<-chan error)
	go func() {
		defer func() {
			close(l.addPlayers)
			close(l.removePlayers)
			for _, gmh := range games {
				gmh.done <- struct{}{}
			}
			for _, smh := range sockets {
				smh.done <- struct{}{}
				smh.done <- struct{}{}
			}
		}()
		for {
			select {
			case <-done:
				return
			case ps := <-l.addPlayers:
				socketMessages, errs = l.addSocket(ps, sockets)
			case pn := <-l.removePlayers:
				l.removeSocket(pn, sockets)
			case m := <-socketMessages:
				var err error
				switch m.Type {
				case game.Create:
					gameMessages, err = l.createGame(m, games)
				case game.Delete:
					err = l.deleteGame(m, games)
				}
				if err == nil {
					l.sendGameMessage(m, games)
				}
				if err != nil {
					l.sendSocketErrorMessage(m, err.Error(), sockets)
				}
			case m := <-gameMessages:
				l.sendSocketMessage(m, sockets)
			case err := <-errs:
				l.log.Printf(err.Error())
			}
		}
	}()
}

func mergeMessages(messages []<-chan game.Message) <-chan game.Message {
	out := make(chan game.Message)
	var wg sync.WaitGroup
	wg.Add(len(messages))
	for _, c := range messages {
		go func(c <-chan game.Message) {
			for v := range c {
				out <- v
			}
			wg.Done()
		}(c)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func mergeErrors(errors ...<-chan error) <-chan error {
	out := make(chan error)
	var wg sync.WaitGroup
	wg.Add(len(errors))
	for _, c := range errors {
		go func(c <-chan error) {
			for v := range c {
				out <- v
			}
			wg.Done()
		}(c)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func (l *Lobby) createGame(m game.Message, games map[game.ID]messageHandler) (<-chan game.Message, error) {
	if len(games) >= l.maxGames {
		return nil, fmt.Errorf("the maximum number of games have already been created")
	}
	var id game.ID = 1
	for existingID := range games {
		if existingID != id {
			break
		}
		id++
	}
	g := l.gameCfg.NewGame(id)
	done := make(chan struct{}, 2)
	writeMessages := make(chan game.Message)
	readMessages := g.Run(done, writeMessages)
	gmh := messageHandler{
		done:          done,
		writeMessages: writeMessages,
		readMessages:  readMessages,
	}
	games[id] = gmh
	messages := make([]<-chan game.Message, len(games))
	i := 0
	for _, g := range games {
		messages[i] = g.readMessages
		i++
	}
	gamesC := mergeMessages(messages)
	return gamesC, nil
}

func (l *Lobby) deleteGame(m game.Message, games map[game.ID]messageHandler) error {
	gmh, ok := games[m.GameID]
	if !ok {
		return fmt.Errorf("no game to delete with id %v", m.GameID)
	}
	gmh.writeMessages <- m
	delete(games, m.GameID)
	return nil
}

func (l *Lobby) handleGameInfos(m game.Message, sockets, games map[game.PlayerName]messageHandler) error {
	smh, ok := sockets[m.PlayerName]
	if !ok {
		return fmt.Errorf("no socket to send game infos to for playerName=%v", m.PlayerName)
	}
	n := len(games)
	c := make(chan game.Info, n)
	for _, gmh := range games {
		gmh.writeMessages <- game.Message{
			Type:         game.Infos,
			PlayerName:   m.PlayerName,
			GameInfoChan: c,
		}
	}
	infos := make([]game.Info, n)
	if n > 0 {
		i := 0
		for {
			gameInfo := <-c
			infos[i] = gameInfo
			i++
			if i == n {
				break
			}
		}
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].CreatedAt < infos[j].CreatedAt
	})
	smh.writeMessages <- game.Message{
		Type:      game.Infos,
		GameInfos: infos,
	}
	return nil
}

// addSocket adds the playerSocket to the socket messageHandlers and returns the merged inbound message and error channels
func (l *Lobby) addSocket(ps playerSocket, sockets map[game.PlayerName]messageHandler) (<-chan game.Message, <-chan error) {
	done := make(chan struct{}, 2)
	readMessages, readErrs := ps.Socket.ReadMessages(done)
	writeMessages, writeErrs := ps.Socket.WriteMessages(done)
	combinedErrs := mergeErrors(readErrs, writeErrs)
	mh := messageHandler{
		done:          done,
		writeMessages: writeMessages,
		readMessages:  readMessages,
		readErrs:      combinedErrs,
	}
	if _, ok := sockets[ps.PlayerName]; ok {
		l.log.Printf("message handler for %v already exists, replacing", ps.PlayerName)
		l.removeSocket(ps.PlayerName, sockets)
	}
	sockets[ps.PlayerName] = mh
	n := len(sockets)
	messages := make([]<-chan game.Message, n)
	errs := make([]<-chan error, n)
	i := 0
	for _, mh := range sockets {
		messages[i] = mh.readMessages
		errs[i] = mh.readErrs
		i++
	}
	messagesC := mergeMessages(messages)
	errsC := mergeErrors(errs...)
	return messagesC, errsC
}

// removeSocket removes a socket from the messageHandlers
func (l *Lobby) removeSocket(pn game.PlayerName, sockets map[game.PlayerName]messageHandler) {
	mh, ok := sockets[pn]
	if !ok {
		l.log.Printf("no socket to remove for %v", pn)
		return
	}
	mh.done <- struct{}{} // readMessages
	mh.done <- struct{}{} // writeMessages
}

// sends a message to the game with the id specified in the message's GameID field
func (l *Lobby) sendGameMessage(m game.Message, games map[game.ID]messageHandler) error {
	g, ok := games[m.GameID]
	if !ok {
		return fmt.Errorf("no game with id %v, please refresh games", m.GameID)
	}
	g.writeMessages <- m
	return nil
}

func (l *Lobby) sendSocketMessage(m game.Message, sockets map[game.PlayerName]messageHandler) {
	mh, ok := sockets[m.PlayerName]
	if !ok {
		l.log.Printf("no socket for player named '%v' to send message to: %v", m.PlayerName, m)
	}
	mh.writeMessages <- m
}

// sendSocketErrorMessage sends the info to the player of the specified message
func (l *Lobby) sendSocketErrorMessage(m game.Message, info string, sockets map[game.PlayerName]messageHandler) {
	mh, ok := sockets[m.PlayerName]
	if !ok {
		l.log.Printf("no socket for player named '%v' to send message to: %v", m.PlayerName, m)
	}
	mh.writeMessages <- game.Message{
		Type:       game.SocketError,
		PlayerName: m.PlayerName,
		Info:       info,
	}
}

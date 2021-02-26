package main

import (
	"context"
	crypto_rand "crypto/rand"
	"fmt"
	"io"
	"log"
	"math/rand"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/db"
	"github.com/jacobpatterson1549/selene-bananas/db/sql"
	"github.com/jacobpatterson1549/selene-bananas/db/user"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
	"github.com/jacobpatterson1549/selene-bananas/game/word"
	"github.com/jacobpatterson1549/selene-bananas/server"
	"github.com/jacobpatterson1549/selene-bananas/server/auth"
	"github.com/jacobpatterson1549/selene-bananas/server/certificate"
	gameController "github.com/jacobpatterson1549/selene-bananas/server/game"
	"github.com/jacobpatterson1549/selene-bananas/server/game/lobby"
	playerController "github.com/jacobpatterson1549/selene-bananas/server/game/player"
	"github.com/jacobpatterson1549/selene-bananas/server/game/socket"
	_ "github.com/lib/pq" // register "postgres" database driver from package init() function
)

// newServer creates the server.
func (m mainFlags) newServer(ctx context.Context, log *log.Logger, db db.Database, wordsFile io.Reader, e embeddedData) (*server.Server, error) {
	timeFunc := func() int64 {
		return time.Now().UTC().Unix()
	}
	key := make([]byte, 64)
	if _, err := crypto_rand.Reader.Read(key); err != nil {
		return nil, fmt.Errorf("generating Tokenizer key: %w", err)
	}
	tokenizerCfg := m.tokenizerConfig(timeFunc)
	tokenizer, err := tokenizerCfg.NewTokenizer(key)
	if err != nil {
		return nil, fmt.Errorf("creating authentication tokenizer: %w", err)
	}
	userDao, err := user.NewDao(db)
	if err != nil {
		return nil, fmt.Errorf("creating user dao: %w", err)
	}
	socketRunnerCfg := m.socketRunnerConfig(timeFunc)
	socketRunner, err := socketRunnerCfg.NewRunner(log)
	if err != nil {
		return nil, fmt.Errorf("creating socket runner: %w", err)
	}
	wordChecker := word.NewChecker(wordsFile)
	gameRunnerCfg := m.gameRunnerConfig(timeFunc)
	gameRunner, err := gameRunnerCfg.NewRunner(log, wordChecker, userDao)
	if err != nil {
		return nil, fmt.Errorf("creating game runner: %w", err)
	}
	lobbyCfg := m.lobbyConfig()
	lobby, err := lobbyCfg.NewLobby(log, socketRunner, gameRunner)
	if err != nil {
		return nil, fmt.Errorf("creating lobby: %w", err)
	}
	challenge := certificate.Challenge{
		Token: m.challengeToken,
		Key:   m.challengeKey,
	}
	colorConfig := m.colorConfig()
	cfg := server.Config{
		HTTPPort:      m.httpPort,
		HTTPSPort:     m.httpsPort,
		StopDur:       20 * time.Second, // should be longer than the PingPeriod of sockets so they can close gracefully
		CacheSec:      m.cacheSec,
		Version:       e.Version,
		TLSCertFile:   m.tlsCertFile,
		TLSKeyFile:    m.tlsKeyFile,
		ColorConfig:   colorConfig,
		NoTLSRedirect: m.noTLSRedirect,
	}
	p := server.Parameters{
		Log:        log,
		Tokenizer:  tokenizer,
		UserDao:    userDao,
		Lobby:      lobby,
		Challenge:  challenge,
		StaticFS:   e.StaticFS,
		TemplateFS: e.TemplateFS,
	}
	return cfg.NewServer(p)
}

// colorConfig creates the color config for the css.
func (mainFlags) colorConfig() server.ColorConfig {
	cc := server.ColorConfig{
		CanvasPrimary: "#000000",
		CanvasDrag:    "#0000ff",
		CanvasTile:    "#f0d0b5",
		LogError:      "#ff0000",
		LogWarning:    "#ff8000",
		LogChat:       "#008000",
		TabBackground: "#ffffc2",
		TableStripe:   "#d9da9c",
		Button:        "#eeeeee",
		ButtonHover:   "#dddddd",
		ButtonActive:  "#cccccc",
	}
	return cc
}

// tokenizerConfig creates the configuration for authentication token reader/writer.
func (mainFlags) tokenizerConfig(timeFunc func() int64) auth.TokenizerConfig {
	oneDay := 24 * time.Hour.Seconds()
	cfg := auth.TokenizerConfig{
		TimeFunc: timeFunc,
		ValidSec: int64(oneDay),
	}
	return cfg
}

// sqlDatabase creates the configuration for a SQL database to persist user information.
func (m mainFlags) sqlDatabaseConfig() sql.DatabaseConfig {
	cfg := sql.DatabaseConfig{
		DriverName:  "postgres",
		DatabaseURL: m.databaseURL,
		QueryPeriod: 5 * time.Second,
	}
	return cfg
}

// lobbyConfig creates the configuration for running and managing players of games.
func (m mainFlags) lobbyConfig() lobby.Config {
	cfg := lobby.Config{
		Debug: m.debugGame,
	}
	return cfg
}

// gameRunnerConfig creates the configuration for running and managing games.
func (m mainFlags) gameRunnerConfig(timeFunc func() int64) gameController.RunnerConfig {
	gameCfg := m.gameConfig(timeFunc)
	cfg := gameController.RunnerConfig{
		Debug:      m.debugGame,
		MaxGames:   4,
		GameConfig: gameCfg,
	}
	return cfg
}

// gameConfig creates the base configuration for all games.
func (m mainFlags) gameConfig(timeFunc func() int64) gameController.Config {
	playerCfg := playerController.Config{
		WinPoints: 10,
	}
	shuffleUnusedTilesFunc := func(tiles []tile.Tile) {
		rand.Shuffle(len(tiles), func(i, j int) {
			tiles[i], tiles[j] = tiles[j], tiles[i]
		})
	}
	shufflePlayersFunc := func(sockets []player.Name) {
		rand.Shuffle(len(sockets), func(i, j int) {
			sockets[i], sockets[j] = sockets[j], sockets[i]
		})
	}
	cfg := gameController.Config{
		Debug:                  m.debugGame,
		TimeFunc:               timeFunc,
		MaxPlayers:             6,
		PlayerCfg:              playerCfg,
		NumNewTiles:            21,
		TileLetters:            "", // 144 default tiles = 144-6*21 = 18 tiles left, which leaves a maximum of 3 snags
		IdlePeriod:             60 * time.Minute,
		ShuffleUnusedTilesFunc: shuffleUnusedTilesFunc,
		ShufflePlayersFunc:     shufflePlayersFunc,
	}
	return cfg
}

// socketRunnerConfig creates the configuration for creating new sockets (each tab that is connected to the lobby).
func (m mainFlags) socketRunnerConfig(timeFunc func() int64) socket.RunnerConfig {
	socketCfg := socket.Config{
		Debug:          m.debugGame,
		TimeFunc:       timeFunc,
		ReadWait:       60 * time.Second,
		WriteWait:      10 * time.Second,
		PingPeriod:     15 * time.Second,
		HTTPPingPeriod: 10 * time.Minute,
	}
	cfg := socket.RunnerConfig{
		Debug:            m.debugGame,
		MaxSockets:       32,
		MaxPlayerSockets: 5,
		SocketConfig:     socketCfg,
	}
	return cfg
}

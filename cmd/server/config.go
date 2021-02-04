package main

import (
	"bufio"
	"context"
	crypto_rand "crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
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
	"github.com/jacobpatterson1549/selene-bananas/server/game"
	gameController "github.com/jacobpatterson1549/selene-bananas/server/game"
	"github.com/jacobpatterson1549/selene-bananas/server/game/lobby"
	playerController "github.com/jacobpatterson1549/selene-bananas/server/game/player"
	"github.com/jacobpatterson1549/selene-bananas/server/game/socket"
	_ "github.com/lib/pq" // register "postgres" database driver from package init() function
)

// serverConfig creates the server configuration.
func serverConfig(ctx context.Context, m mainFlags, log *log.Logger) (*server.Config, error) {
	timeFunc := func() int64 {
		return time.Now().UTC().Unix()
	}
	keyReader := crypto_rand.Reader
	tokenizerCfg := tokenizerConfig(keyReader, timeFunc)
	tokenizer, err := tokenizerCfg.NewTokenizer()
	if err != nil {
		return nil, fmt.Errorf("creating authentication tokenizer: %w", err)
	}
	if len(m.databaseURL) == 0 {
		return nil, fmt.Errorf("missing data-source uri")
	}
	sqlDB, err := sqlDatabase(m)
	if err != nil {
		return nil, fmt.Errorf("creating SQL database: %w", err)
	}
	userDaoCfg := userDaoConfig(sqlDB)
	ud, err := userDaoCfg.NewDao()
	if err != nil {
		return nil, fmt.Errorf("creating user dao: %w", err)
	}
	if err = ud.Setup(ctx); err != nil {
		return nil, fmt.Errorf("setting up user dao: %w", err)
	}
	socketRunnerConfig := socketRunnerConfig(m, log, timeFunc)
	socketRunner, err := socketRunnerConfig.NewRunner()
	if err != nil {
		return nil, fmt.Errorf("creating socket runner: %w", err)
	}
	gameRunnerConfig, err := gameRunnerConfig(m, log, ud, timeFunc)
	if err != nil {
		return nil, fmt.Errorf("creating game runner config: %w", err)
	}
	gameRunner, err := gameRunnerConfig.NewRunner()
	if err != nil {
		return nil, fmt.Errorf("creating game runner: %w", err)
	}
	lobbyCfg := lobbyConfig(log)
	lobby, err := lobbyCfg.NewLobby(socketRunner, gameRunner)
	if err != nil {
		return nil, fmt.Errorf("creating lobby: %w", err)
	}
	v, err := version(m)
	if err != nil {
		return nil, fmt.Errorf("creating build version: %w", err)
	}
	c := certificate.Challenge{
		Token: m.challengeToken,
		Key:   m.challengeKey,
	}
	cc := colorConfig()
	cfg := server.Config{
		HTTPPort:      m.httpPort,
		HTTPSPort:     m.httpsPort,
		Log:           log,
		Tokenizer:     tokenizer,
		UserDao:       ud,
		Lobby:         lobby,
		StopDur:       time.Second,
		CacheSec:      m.cacheSec,
		Version:       v,
		Challenge:     c,
		TLSCertFile:   m.tlsCertFile,
		TLSKeyFile:    m.tlsKeyFile,
		ColorConfig:   cc,
		NoTLSRedirect: m.noTLSRedirect,
	}
	return &cfg, nil
}

// colorConfig creates the color config for the css.
func colorConfig() server.ColorConfig {
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
func tokenizerConfig(keyReader io.Reader, timeFunc func() int64) auth.TokenizerConfig {
	var tokenValidDurationSec int64 = int64((24 * time.Hour).Seconds()) // 1 day
	cfg := auth.TokenizerConfig{
		KeyReader: keyReader,
		TimeFunc:  timeFunc,
		ValidSec:  tokenValidDurationSec,
	}
	return cfg
}

// sqlDatabase creates a SQL database to persist user information.
func sqlDatabase(m mainFlags) (db.Database, error) {
	cfg := sql.DatabaseConfig{
		DriverName:  "postgres",
		DatabaseURL: m.databaseURL,
		QueryPeriod: 5 * time.Second,
	}
	return cfg.NewDatabase()
}

// userDaoConfig creates a user dao configuration.
func userDaoConfig(d db.Database) user.DaoConfig {
	cfg := user.DaoConfig{
		DB:           d,
		ReadFileFunc: ioutil.ReadFile,
	}
	return cfg
}

// lobbyConfig creates the configuration for running and managing players of games.
func lobbyConfig(log *log.Logger) lobby.Config {
	cfg := lobby.Config{
		Log: log,
	}
	return cfg
}

// gameRunnerConfig creates the configuration for running and managing games.
func gameRunnerConfig(m mainFlags, log *log.Logger, ud *user.Dao, timeFunc func() int64) (*game.RunnerConfig, error) {
	gameCfg, err := gameConfig(m, log, ud, timeFunc)
	if err != nil {
		return nil, fmt.Errorf("creating game config: %w", err)
	}
	cfg := game.RunnerConfig{
		Log:        log,
		MaxGames:   4,
		GameConfig: *gameCfg,
	}
	return &cfg, nil
}

// gameConfig creates the base configuration for all games.
func gameConfig(m mainFlags, log *log.Logger, ud *user.Dao, timeFunc func() int64) (*gameController.Config, error) {
	wordsFile, err := os.Open(m.wordsFile)
	if err != nil {
		return nil, fmt.Errorf("trying to open words file: %w", err)
	}
	playerCfg := playerController.Config{
		WinPoints: 10,
	}
	wordChecker := word.NewChecker(wordsFile)
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
		Log:                    log,
		TimeFunc:               timeFunc,
		UserDao:                ud,
		MaxPlayers:             8,
		PlayerCfg:              playerCfg,
		NumNewTiles:            21,
		TileLetters:            "",
		WordChecker:            wordChecker,
		IdlePeriod:             60 * time.Minute,
		ShuffleUnusedTilesFunc: shuffleUnusedTilesFunc,
		ShufflePlayersFunc:     shufflePlayersFunc,
	}
	return &cfg, nil
}

// socketRunnerConfig creates the configuration for creating new sockets (each tab that is connected to the lobby).
func socketRunnerConfig(m mainFlags, log *log.Logger, timeFunc func() int64) socket.RunnerConfig {
	socketCfg := socket.Config{
		Debug:               m.debugGame,
		Log:                 log,
		ReadWait:            60 * time.Second,
		WriteWait:           10 * time.Second,
		PingPeriod:          54 * time.Second, // readWait * 0.9
		ActivityCheckPeriod: 10 * time.Minute,
	}
	cfg := socket.RunnerConfig{
		Log:              log,
		MaxSockets:       32,
		MaxPlayerSockets: 5,
		SocketConfig:     socketCfg,
	}
	return cfg
}

// version reads the first word of the versionFile to use as the version.
func version(m mainFlags) (string, error) {
	versionFile, err := os.Open(m.versionFile)
	if err != nil {
		return "", fmt.Errorf("trying to open version file: %v", err)
	}
	scanner := bufio.NewScanner(versionFile)
	scanner.Split(bufio.ScanWords)
	if !scanner.Scan() {
		return "", fmt.Errorf("no words in version file")
	}
	return scanner.Text(), nil
}

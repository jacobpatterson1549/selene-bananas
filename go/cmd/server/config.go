package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/go/db"
	"github.com/jacobpatterson1549/selene-bananas/go/game"
	"github.com/jacobpatterson1549/selene-bananas/go/game/controller"
	"github.com/jacobpatterson1549/selene-bananas/go/game/lobby"
	"github.com/jacobpatterson1549/selene-bananas/go/game/socket"
	"github.com/jacobpatterson1549/selene-bananas/go/game/tile"
	"github.com/jacobpatterson1549/selene-bananas/go/server"
	_ "github.com/lib/pq" // regester "postgres" database driver from package init() function
)

func serverConfig(ctx context.Context, m mainFlags, log *log.Logger) (*server.Config, error) {
	timeFunc := func() int64 {
		return time.Now().UTC().Unix()
	}
	rand := rand.New(rand.NewSource(timeFunc()))
	tokenizerCfg := tokenizerConfig(rand, timeFunc)
	tokenizer, err := tokenizerCfg.NewTokenizer()
	if err != nil {
		return nil, err
	}
	if len(m.databaseURL) == 0 {
		return nil, fmt.Errorf("missing data-source uri")
	}
	d, err := db.NewPostgresDatabase(m.databaseURL)
	if err != nil {
		return nil, err
	}
	userDaoCfg := userDaoConfig(d)
	ud, err := userDaoCfg.NewUserDao()
	if err != nil {
		return nil, err
	}
	err = ud.Setup(ctx)
	if err != nil {
		return nil, err
	}
	lobbyCfg, err := lobbyConfig(m, log, rand, ud, timeFunc)
	if err != nil {
		return nil, err
	}
	cfg := server.Config{
		AppName:   m.applicationName,
		Port:      m.serverPort,
		Log:       log,
		Tokenizer: tokenizer,
		UserDao:   ud,
		LobbyCfg:  *lobbyCfg,
		StopDur:   time.Second,
		CacheSec:  m.cacheSec,
	}
	return &cfg, nil
}

func tokenizerConfig(rand *rand.Rand, timeFunc func() int64) server.TokenizerConfig {
	var tokenValidDurationSec int64 = int64((24 * time.Hour).Seconds()) // 1 day
	cfg := server.TokenizerConfig{
		Rand:     rand,
		TimeFunc: timeFunc,
		ValidSec: tokenValidDurationSec,
	}
	return cfg
}

func userDaoConfig(d db.Database) db.UserDaoConfig {
	cfg := db.UserDaoConfig{
		DB:          d,
		QueryPeriod: 5 * time.Second,
	}
	return cfg
}

func lobbyConfig(m mainFlags, log *log.Logger, rand *rand.Rand, ud *db.UserDao, timeFunc func() int64) (*lobby.Config, error) {
	gameCfg, err := gameConfig(m, log, rand, ud, timeFunc)
	socketCfg := socketConfig(m, log, timeFunc)
	if err != nil {
		return nil, err
	}
	cfg := lobby.Config{
		Debug:      m.debugGame,
		Log:        log,
		MaxGames:   4,
		MaxSockets: 32,
		GameCfg:    *gameCfg,
		SocketCfg:  socketCfg,
	}
	return &cfg, nil
}

func gameConfig(m mainFlags, log *log.Logger, rand *rand.Rand, ud *db.UserDao, timeFunc func() int64) (*controller.Config, error) {
	wordsFile, err := os.Open(m.wordsFile)
	if err != nil {
		return nil, err
	}
	wc, err := game.NewWordChecker(wordsFile)
	if err != nil {
		return nil, err
	}
	shuffleUnusedTilesFunc := func(tiles []tile.Tile) {
		rand.Shuffle(len(tiles), func(i, j int) {
			tiles[i], tiles[j] = tiles[j], tiles[i]
		})
	}
	shufflePlayersFunc := func(sockets []game.PlayerName) {
		rand.Shuffle(len(sockets), func(i, j int) {
			sockets[i], sockets[j] = sockets[j], sockets[i]
		})
	}
	cfg := controller.Config{
		Debug:                  m.debugGame,
		Log:                    log,
		TimeFunc:               timeFunc,
		UserDao:                ud,
		MaxPlayers:             8,
		NumNewTiles:            21,
		TileLetters:            "",
		Words:                  *wc,
		IdlePeriod:             60 * time.Minute,
		ShuffleUnusedTilesFunc: shuffleUnusedTilesFunc,
		ShufflePlayersFunc:     shufflePlayersFunc,
	}
	return &cfg, nil
}

func socketConfig(m mainFlags, log *log.Logger, timeFunc func() int64) socket.Config {
	cfg := socket.Config{
		Debug:          m.debugGame,
		Log:            log,
		TimeFunc:       timeFunc,
		PongPeriod:     20 * time.Second,
		PingPeriod:     16 * time.Second,
		IdlePeriod:     15 * time.Minute,
		HTTPPingPeriod: 10 * time.Minute,
	}
	return cfg
}

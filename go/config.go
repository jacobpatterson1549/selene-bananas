package main

import (
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
	_ "github.com/lib/pq"
)

func serverConfig(m mainFlags, log *log.Logger) (*server.Config, error) {
	rand := rand.New(rand.NewSource(time.Now().Unix()))
	tokenizerCfg := tokenizerConfig(rand)
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
	ud := db.NewUserDao(d)
	err = ud.Setup()
	if err != nil {
		return nil, err
	}
	lobbyCfg, err := lobbyConfig(m, log, rand, ud)
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
	}
	return &cfg, nil
}

func tokenizerConfig(rand *rand.Rand) server.TokenizerConfig {
	var tokenValidDurationSec int64 = 365 * 24 * 60 * 60 // 1 year
	cfg := server.TokenizerConfig{
		Rand:     rand,
		ValidSec: tokenValidDurationSec,
	}
	return cfg
}

func lobbyConfig(m mainFlags, log *log.Logger, rand *rand.Rand, ud db.UserDao) (*lobby.Config, error) {
	gameCfg, err := gameConfig(m, log, rand, ud)
	socketCfg := socketConfig(m, log)
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

func gameConfig(m mainFlags, log *log.Logger, rand *rand.Rand, ud db.UserDao) (*controller.Config, error) {
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

func socketConfig(m mainFlags, log *log.Logger) socket.Config {
	cfg := socket.Config{
		Debug:          m.debugGame,
		Log:            log,
		PongPeriod:     20 * time.Second,
		PingPeriod:     15 * time.Second,
		IdlePeriod:     15 * time.Minute,
		HTTPPingPeriod: 10 * time.Minute,
	}
	return cfg
}

package socket

import (
	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
)

type mockUser string

func (m mockUser) JWT() string {
	return string(m)
}

func (m mockUser) Username() string {
	return string(m)
}

func (mockUser) Logout() {
	// NOOP
}

type mockGame game.ID

func (m mockGame) ID() game.ID {
	return game.ID(m)
}

func (mockGame) Leave() {
	// NOOP
}

func (mockGame) UpdateInfo(m message.Message) {
	// NOOP
}

type mockLobby struct {
	SetGameInfosFunc func(gameInfos []game.Info, username string)
}

func (m mockLobby) SetGameInfos(gameInfos []game.Info, username string) {
	m.SetGameInfosFunc(gameInfos, username)
}

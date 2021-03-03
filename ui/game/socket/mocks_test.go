package socket

import (
	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
)

type mockUser string

func (m mockUser) JWT() string {
	return string(m)
}

func (mockUser) Username() string {
	return ""
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

type mockLobby struct{}

func (mockLobby) SetGameInfos(gameInfos []game.Info, username string) {
	// NOOP
}

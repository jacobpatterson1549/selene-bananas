package socket

import (
	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
)

type mockUser string

func (u mockUser) JWT() string {
	return string(u)
}

func (u mockUser) Username() string {
	return ""
}

func (u *mockUser) Logout() {
	// NOOP
}

type mockGame game.ID

func (g mockGame) ID() game.ID {
	return game.ID(g)
}

func (g mockGame) Leave() {
	// NOOP
}

func (g mockGame) UpdateInfo(m message.Message) {
	// NOOP
}

type mockLobby struct{}

func (l mockLobby) SetGameInfos(gameInfos []game.Info, username string) {
	// NOOP
}

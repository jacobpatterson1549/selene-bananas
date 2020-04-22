package game

import (
	"encoding/json"
	"fmt"
)

type (
	// MessageType is an enumeration of supported messages
	MessageType int
	// Message is passed between websocket connections
	Message struct {
		Type    MessageType     `json:"type"`
		Command json.RawMessage `json:"command"`
	}
	// UserLoginCommand contains information to login a user
	UserLoginCommand struct {
		Username string
		Password string
	}
	// UserUpdatePasswordCommand contains information to update a user's password
	UserUpdatePasswordCommand struct {
		Username    string
		OldPassword string
		NewPassword string
	}
	// UserDeleteCommand contains information to delete a user
	UserDeleteCommand struct {
		Password string
	}
)

const (
	userLogin MessageType = iota + 1
	userLogout
	userUpdatePassword
	userDelete
	userClose
	gameCreate
	gameStart
	gameSnag
	gameSwap
	gameFinish
	gameClose
)

func (m Message) handle() error {
	switch m.Type {
	//TODO
	// userLogin
	// userLogout
	// userUpdatePassword
	// userDelete
	// userClose
	// gameCreate
	// gameStart
	// gameSnag
	// gameSwap
	// gameFinish
	// gameClose
	default:
		return fmt.Errorf("unknown message type: %v", m.Type)
	}
}

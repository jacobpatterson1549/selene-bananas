package server

import (
	"fmt"
	"net/http"

	"github.com/jacobpatterson1549/selene-bananas/go/db"
	"github.com/jacobpatterson1549/selene-bananas/go/game"
)

type (
	userChangeFn func(r *http.Request) error
)

func (s Server) handleUserCreate(w http.ResponseWriter, r *http.Request) error {
	err := r.ParseForm()
	if err != nil {
		return fmt.Errorf("parsing form: %w", err)
	}
	username := r.FormValue("username")
	password := r.FormValue("password_confirm")
	u, err := db.NewUser(username, password)
	if err != nil {
		return err
	}
	err = s.userDao.Create(u)
	if err != nil {
		return err
	}
	return nil
}

func (s Server) handleUserLogin(w http.ResponseWriter, r *http.Request) error {
	err := r.ParseForm()
	if err != nil {
		return fmt.Errorf("parsing form: %w", err)
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	u, err := db.NewUser(username, password)
	if err != nil {
		return err
	}
	u2, err := s.userDao.Read(u)
	if err != nil {
		s.log.Printf("login failure: %v", err)
		http.Error(w, "incorrect username/password", http.StatusUnauthorized)
		return nil
	}
	return s.addAuthorization(w, u2)
}

func (s Server) handleUserJoinLobby(w http.ResponseWriter, r *http.Request, username string) error {
	playerName := game.PlayerName(username)
	err := s.lobby.AddUser(playerName, w, r)
	if err != nil {
		return fmt.Errorf("websocket error: %w", err)
	}
	return nil
}

func (s Server) handleUserUpdatePassword(w http.ResponseWriter, r *http.Request, username string) error {
	password := r.FormValue("password")
	newPassword := r.FormValue("password_confirm")
	u, err := db.NewUser(username, password)
	if err != nil {
		return err
	}
	err = s.userDao.UpdatePassword(u, newPassword)
	if err != nil {
		return err
	}
	playerName := game.PlayerName(username)
	s.lobby.RemoveUser(playerName)
	return nil
}

func (s Server) handleUserDelete(w http.ResponseWriter, r *http.Request, username string) error {
	password := r.FormValue("password")
	u, err := db.NewUser(username, password)
	if err != nil {
		return err
	}
	err = s.userDao.Delete(u)
	if err != nil {
		return err
	}
	playerName := game.PlayerName(username)
	s.lobby.RemoveUser(playerName)
	return err
}

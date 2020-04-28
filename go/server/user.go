package server

import (
	"fmt"
	"net/http"

	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
)

type (
	userChangeFn func(r *http.Request) error
)

func (s server) handleUserCreate(w http.ResponseWriter, r *http.Request) error {
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

func (s server) handleUserLogin(w http.ResponseWriter, r *http.Request) error {
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

func (s server) handleUserJoinLobby(w http.ResponseWriter, r *http.Request, tokenUsername db.Username) error {
	err := s.lobby.AddUser(tokenUsername, w, r)
	if err != nil {
		return fmt.Errorf("websocket error: %w", err)
	}
	return nil
}

func (s server) handleUserUpdatePassword(w http.ResponseWriter, r *http.Request, tokenUsername db.Username) error {
	err := r.ParseForm()
	if err != nil {
		return fmt.Errorf("parsing form: %w", err)
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	newPassword := r.FormValue("password_confirm")
	if string(tokenUsername) != username {
		return fmt.Errorf("cannot modify other user") // TODO: return 403 forbidden
	}
	u, err := db.NewUser(username, password)
	if err != nil {
		return err
	}
	err = s.userDao.UpdatePassword(u, newPassword)
	if err != nil {
		return err
	}
	s.lobby.RemoveUser(tokenUsername)
	return nil
}

func (s server) handleUserDelete(w http.ResponseWriter, r *http.Request, tokenUsername db.Username) error {
	err := r.ParseForm()
	if err != nil {
		return fmt.Errorf("parsing form: %w", err)
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	if string(tokenUsername) != username {
		return fmt.Errorf("cannot modify other user") // TODO: return 403 forbidden
	}
	u, err := db.NewUser(username, password)
	if err != nil {
		return err
	}
	err = s.userDao.Delete(u)
	if err != nil {
		return err
	}
	s.lobby.RemoveUser(u.Username)
	return err
}

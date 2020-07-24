package server

import (
	"fmt"
	"net/http"

	"github.com/jacobpatterson1549/selene-bananas/db"
)

// handleUserCreate creates a user, adding it to the database.
func (s Server) handleUserCreate(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password_confirm")
	u, err := db.NewUser(username, password)
	if err != nil {
		s.handleError(w, err)
		return
	}
	ctx := r.Context()
	if err := s.userDao.Create(ctx, *u); err != nil {
		s.handleError(w, err)
		return
	}
}

// handleUserLogin signs a user in, writing the token to the response.
func (s Server) handleUserLogin(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	u, err := db.NewUser(username, password)
	if err != nil {
		s.handleError(w, err)
		return
	}
	ctx := r.Context()
	u2, err := s.userDao.Read(ctx, *u)
	if err != nil {
		s.log.Printf("login failure: %v", err)
		http.Error(w, "incorrect username/password", http.StatusUnauthorized)
		return
	}
	token, err := s.tokenizer.Create(u2.Username, u2.Points)
	if err != nil {
		s.handleError(w, err)
		return
	}
	if _, err := w.Write([]byte(token)); err != nil {
		err = fmt.Errorf("writing authorization token: %w", err)
		s.handleError(w, err)
		return
	}
}

// handleUserJoinLobby adds the user to the lobby.
func (s Server) handleUserJoinLobby(w http.ResponseWriter, r *http.Request, username string) {
	if err := s.lobby.AddUser(username, w, r); err != nil {
		err = fmt.Errorf("websocket error: %w", err)
		s.handleError(w, err)
		return
	}
}

// handleUserUpdatePassword updates the user's password.
func (s Server) handleUserUpdatePassword(w http.ResponseWriter, r *http.Request, username string) {
	password := r.FormValue("password")
	newPassword := r.FormValue("password_confirm")
	u, err := db.NewUser(username, password)
	if err != nil {
		s.handleError(w, err)
		return
	}
	ctx := r.Context()
	if err := s.userDao.UpdatePassword(ctx, *u, newPassword); err != nil {
		s.handleError(w, err)
		return
	}
	s.lobby.RemoveUser(username)
}

// handleUserDelete deletes the user from the database.
func (s Server) handleUserDelete(w http.ResponseWriter, r *http.Request, username string) {
	password := r.FormValue("password")
	u, err := db.NewUser(username, password)
	if err != nil {
		s.handleError(w, err)
		return
	}
	ctx := r.Context()
	if err := s.userDao.Delete(ctx, *u); err != nil {
		s.handleError(w, err)
		return
	}
	s.lobby.RemoveUser(username)
}

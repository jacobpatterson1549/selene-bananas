package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/jacobpatterson1549/selene-bananas/db/user"
)

type (
	// UserDao contains CRUD operations for user-related information.
	UserDao interface {
		Create(ctx context.Context, u user.User) error
		Read(ctx context.Context, u user.User) (*user.User, error)
		UpdatePassword(ctx context.Context, u user.User, newP string) error
		Delete(ctx context.Context, u user.User) error
	}

	// Lobby is the place users can create, join, and participate in games.
	Lobby interface {
		Run(ctx context.Context, wg *sync.WaitGroup)
		AddUser(username string, w http.ResponseWriter, r *http.Request) error
		RemoveUser(username string)
	}
)

// handleUserCreate creates a user, adding it to the database.
func (s *Server) handleUserCreate(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password_confirm")
	u, err := user.New(username, password)
	if err != nil {
		s.writeInternalError(w, err)
		return
	}
	ctx := r.Context()
	if err := s.userDao.Create(ctx, *u); err != nil {
		s.writeInternalError(w, err)
		return
	}
}

// handleUserLogin signs a user in, writing the token to the response.
func (s *Server) handleUserLogin(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	u, err := user.New(username, password)
	if err != nil {
		s.writeInternalError(w, err)
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
		s.writeInternalError(w, err)
		return
	}
	if _, err := w.Write([]byte(token)); err != nil {
		err = fmt.Errorf("writing authorization token: %w", err)
		s.writeInternalError(w, err)
		return
	}
}

// handleUserLobby adds the user to the lobby.
func (s *Server) handleUserLobby(w http.ResponseWriter, r *http.Request) {
	tokenString := r.FormValue("access_token")
	username, err := s.tokenizer.ReadUsername(tokenString)
	if err != nil {
		s.log.Printf("reading username from token: %v", err)
		httpError(w, http.StatusUnauthorized)
		return
	}
	if err := s.lobby.AddUser(username, w, r); err != nil {
		err = fmt.Errorf("websocket error: %w", err)
		s.writeInternalError(w, err)
		return
	}
}

// handleUserUpdatePassword updates the user's password.
func (s *Server) handleUserUpdatePassword(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	newPassword := r.FormValue("password_confirm")
	u, err := user.New(username, password)
	if err != nil {
		s.writeInternalError(w, err)
		return
	}
	ctx := r.Context()
	if err := s.userDao.UpdatePassword(ctx, *u, newPassword); err != nil {
		s.writeInternalError(w, err)
		return
	}
	s.lobby.RemoveUser(username)
}

// handleUserDelete deletes the user from the database.
func (s *Server) handleUserDelete(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	u, err := user.New(username, password)
	if err != nil {
		s.writeInternalError(w, err)
		return
	}
	ctx := r.Context()
	if err := s.userDao.Delete(ctx, *u); err != nil {
		s.writeInternalError(w, err)
		return
	}
	s.lobby.RemoveUser(username)
}

package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jacobpatterson1549/selene-bananas/db/user"
	"github.com/jacobpatterson1549/selene-bananas/server/log"
)

// UserDao contains CRUD operations for user-related information.
type UserDao interface {
	Create(ctx context.Context, u user.User) error
	Login(ctx context.Context, u user.User) (*user.User, error)
	UpdatePassword(ctx context.Context, u user.User, newP string) error
	Delete(ctx context.Context, u user.User) error
}

// userCreateHandler creates a user, adding it to the database.
func userCreateHandler(userDao UserDao, log log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.FormValue("username")
		password := r.FormValue("password_confirm")
		u, err := user.New(username, password)
		if err != nil {
			writeInternalError(err, log, w)
			return
		}
		ctx := r.Context()
		if err := userDao.Create(ctx, *u); err != nil {
			writeInternalError(err, log, w)
			return
		}
	}
}

// userLoginHandler signs a user in, writing the token to the response.
func userLoginHandler(userDao UserDao, tokenizer Tokenizer, log log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.FormValue("username")
		password := r.FormValue("password")
		u, err := user.New(username, password)
		if err != nil {
			writeInternalError(err, log, w)
			return
		}
		ctx := r.Context()
		u2, err := userDao.Login(ctx, *u)
		if err != nil {
			if err != user.ErrIncorrectLogin {
				log.Printf("login failure: %v", err)
				writeInternalError(err, log, w)
				return
			}
			http.Error(w, "incorrect username/password", http.StatusUnauthorized)
			return
		}
		token, err := tokenizer.Create(u2.Username, u2.Points)
		if err != nil {
			writeInternalError(err, log, w)
			return
		}
		w.Write([]byte(token))
	}
}

// userLobbyConnectHandler adds the user to the lobby.
func userLobbyConnectHandler(tokenizer Tokenizer, lobby Lobby, log log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.FormValue("access_token")
		username, err := tokenizer.ReadUsername(tokenString)
		if err != nil {
			log.Printf("reading username from token: %v", err)
			http.Error(w, "unauthorized to join lobby, try logging out and in", http.StatusUnauthorized)
			return
		}
		if err := lobby.AddUser(username, w, r); err != nil {
			err = fmt.Errorf("websocket error: %w", err)
			writeInternalError(err, log, w)
			return
		}
	}
}

// userUpdatePasswordHandler updates the user's password.
func userUpdatePasswordHandler(userDao UserDao, lobby Lobby, log log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.FormValue("username")
		password := r.FormValue("password")
		newPassword := r.FormValue("password_confirm")
		u, err := user.New(username, password)
		if err != nil {
			writeInternalError(err, log, w)
			return
		}
		ctx := r.Context()
		if err := userDao.UpdatePassword(ctx, *u, newPassword); err != nil {
			writeInternalError(err, log, w)
			return
		}
		lobby.RemoveUser(username)
	}
}

// userDeleteHandler deletes the user from the database.
func userDeleteHandler(userDao UserDao, lobby Lobby, log log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.FormValue("username")
		password := r.FormValue("password")
		u, err := user.New(username, password)
		if err != nil {
			writeInternalError(err, log, w)
			return
		}
		ctx := r.Context()
		if err := userDao.Delete(ctx, *u); err != nil {
			writeInternalError(err, log, w)
			return
		}
		lobby.RemoveUser(username)
	}
}

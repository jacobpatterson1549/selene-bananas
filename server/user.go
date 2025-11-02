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
	Backend() user.Backend
}

// userCreateHandler creates a user, adding it to the database.
func userCreateHandler(userDao UserDao, log log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.FormValue("username")
		password := r.FormValue("password_confirm")
		u := user.User{
			Username: username,
			Password: password,
		}
		ctx := r.Context()
		if err := userDao.Create(ctx, u); err != nil {
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
		u := user.User{
			Username: username,
			Password: password,
		}
		ctx := r.Context()
		u2, err := userDao.Login(ctx, u)
		if err != nil {
			handleUserDaoError(w, err, "login", log)
			return
		}
		token, err := tokenizer.Create(u2.Username, false, u2.Points)
		if err != nil {
			writeInternalError(err, log, w)
			return
		}
		w.Write([]byte(token))
	}
}

// userLobbyConnectHandler adds the user to the lobby.
func userLobbyConnectHandler(lobby Lobby, tokenizer Tokenizer, log log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		authorization := "Bearer " + query.Get("access_token")
		r = withAuthorization(w, r, authorization, tokenizer, log)

		u, err := getUser(r)
		if err != nil {
			writeInternalError(err, log, w)
			return
		}
		if err := lobby.AddUser(u.Username, w, r); err != nil {
			err = fmt.Errorf("websocket error: %w", err)
			writeInternalError(err, log, w)
			return
		}
	}
}

// userUpdatePasswordHandler updates the user's password.
func userUpdatePasswordHandler(userDao UserDao, lobby Lobby, log log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := getUser(r)
		if err != nil {
			writeInternalError(err, log, w)
			return
		}
		u.Password = r.FormValue("password")
		newPassword := r.FormValue("password_confirm")
		ctx := r.Context()
		if err := userDao.UpdatePassword(ctx, *u, newPassword); err != nil {
			handleUserDaoError(w, err, "update password", log)
			return
		}
		lobby.RemoveUser(u.Username)
	}
}

// userDeleteHandler deletes the user from the database.
func userDeleteHandler(userDao UserDao, e Oauth2Endpoint, lobby Lobby, log log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := getUser(r)
		if err != nil {
			writeInternalError(err, log, w)
			return
		}
		u.Password = r.FormValue("password")
		ctx := r.Context()
		if err := userDao.Delete(ctx, *u); err != nil {
			handleUserDaoError(w, err, "delete", log)
			return
		}
		lobby.RemoveUser(u.Username)

		if u.IsOauth2 {
			accessToken := r.FormValue("access_token")
			if err := e.RevokeAccess(accessToken); err != nil {
				writeInternalError(err, log, w)
				return
			}
		}
	}
}

// handleUserDaoError writes a 401 error for users that signed in incorrectly, otherwise writing and logging an internal server error.
func handleUserDaoError(w http.ResponseWriter, err error, action string, log log.Logger) {
	switch err {
	case user.ErrIncorrectLogin:
		http.Error(w, err.Error(), http.StatusUnauthorized)
	default:
		log.Printf("%user v failure: %v", action, err)
		writeInternalError(err, log, w)
	}
}

func getUser(r *http.Request) (*user.User, error) {
	ctx := r.Context()
	username, ok := ctx.Value(usernameContextKey).(string)
	if !ok {
		return nil, fmt.Errorf("could not get username")
	}
	isOauth2, ok := ctx.Value(isOauth2ContextKey).(bool)
	if !ok {
		return nil, fmt.Errorf("could not get isOauth2")
	}
	u := user.User{
		Username: username,
		IsOauth2: isOauth2,
	}
	return &u, nil
}

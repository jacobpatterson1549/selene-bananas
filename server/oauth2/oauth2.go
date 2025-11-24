// Package oauth2 allows users to sign in using trusted sites
package oauth2

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jacobpatterson1549/selene-bananas/db/user"
	"golang.org/x/oauth2"
)

type (
	UserDao interface {
		Create(ctx context.Context, u user.User) error
		Login(ctx context.Context, u user.User) (*user.User, error)
	}
	Tokenizer interface {
		Create(username string, isOauth2 bool, points int) (string, error)
	}

	Endpoint struct {
		conf      oauth2.Config
		opts      []oauth2.AuthCodeOption
		csrfToken string
	}
	tokenResponse struct {
		Sub string
	}
	auth struct {
		ID          string
		Name        string
		AccessToken string
	}
)

// HandleLogin redirects to a Google Oauth2 login page.
func (e Endpoint) HandleLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := e.csrfToken
		conf := e.conf
		url := conf.AuthCodeURL(state, e.opts...)

		w.Header().Add("", "Access-Control-Allow-Origin")
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	}
}

// HandleCallback initializes the user after requesting the token
func (e Endpoint) HandleCallback(ud UserDao, tokenizer Tokenizer, jwtHandler func(jwt, accessToken string, u user.User) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// authenticate
		a, err := e.authenticate(r)
		if err != nil {
			err = fmt.Errorf("getting oauth2 user: %w", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if len(a.ID) == 0 {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}
		// login
		username := a.Username()
		u := user.User{
			Username: username,
			IsOauth2: true,
		}
		u2, err := ud.Login(r.Context(), u)
		if err == user.ErrIncorrectLogin {
			err = ud.Create(r.Context(), u)
			u2 = &u
		}
		if err != nil {
			err = fmt.Errorf("getting/creating user: %w", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// jwt
		jwt, err := tokenizer.Create(u2.Username, true, u2.Points)
		if err != nil {
			err = fmt.Errorf("creating user token: %w", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h := jwtHandler(jwt, a.AccessToken, *u2)
		h.ServeHTTP(w, r)
	}
}

// RevokeAccess deletes the account
func (e Endpoint) RevokeAccess(accessToken string) error {
	return e.revokeAccess(accessToken)
}

func (a auth) Username() string {
	return "g-" + a.ID
}

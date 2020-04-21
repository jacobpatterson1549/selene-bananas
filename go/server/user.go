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
	err = s.userDao.Create(db.NewUser(username, password))
	if err != nil {
		return err
	}
	handleUserLogout(w)
	return nil
}

func (s server) handleUserLogin(w http.ResponseWriter, r *http.Request) error {
	err := r.ParseForm()
	if err != nil {
		return fmt.Errorf("parsing form: %w", err)
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	u := db.NewUser(username, password)
	u2, err := s.userDao.Read(u)
	s.userDao.Read(db.NewUser(username, password))
	if err != nil {
		return err
	}

	// TODO: open lobby connection
	// err = s.lobby.AddUser(u2, w, r)
	// if err != nil {
	// 	return fmt.Errorf("websocket error: %w", err)
	// }

	token, err := s.tokenizer.Create(u2)
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(token))
	if err != nil {
		return fmt.Errorf("writing token: %w", err)
	}
	return nil
}

func (s server) handleUserUpdatePassword(r *http.Request) error {
	err := r.ParseForm()
	if err != nil {
		return fmt.Errorf("parsing form: %w", err)
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	newPassword := r.FormValue("password_confirm")
	u := db.NewUser(username, password)
	return s.userDao.UpdatePassword(u, newPassword)
}

func (s server) handleUserDelete(r *http.Request) error {
	err := r.ParseForm()
	if err != nil {
		return fmt.Errorf("parsing form: %w", err)
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	u := db.NewUser(username, password)
	err = s.userDao.Delete(u)
	s.lobby.RemoveUser(u) // ignore result
	return err
}

func handleUserLogout(w http.ResponseWriter) {
	w.WriteHeader(http.StatusSeeOther)
	w.Header().Set("Location", "/")
}

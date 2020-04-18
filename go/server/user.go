package server

import (
	"fmt"
	"net/http"

	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
)

type (
	userChangeFn func(r *http.Request) error
)

func (s server) handleUserCreate(r *http.Request) error {
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

	err = s.lobby.AddUser(u2, w, r)
	if err != nil {
		return fmt.Errorf("websocket error: %w", err)
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
	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusSeeOther)
}

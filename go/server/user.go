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
	u, err := db.NewUser(username, password)
	if err != nil {
		return err
	}
	u2, err := s.userDao.Read(u)
	if err != nil {
		return err
	}

	// TODO: open lobby connection
	// err = s.lobby.AddUser(u2, w, r)
	// if err != nil {
	// 	return fmt.Errorf("websocket error: %w", err)
	// }

	return s.addAuthorization(w, u2)
}

func (s server) handleUserUpdatePassword(w http.ResponseWriter, r *http.Request) error {
	err := r.ParseForm()
	if err != nil {
		return fmt.Errorf("parsing form: %w", err)
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	newPassword := r.FormValue("password_confirm")
	u, err := db.NewUser(username, password)
	if err != nil {
		return err
	}
	return s.userDao.UpdatePassword(u, newPassword)
}

func (s server) handleUserDelete(w http.ResponseWriter, r *http.Request) error {
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
	err = s.userDao.Delete(u)
	s.lobby.RemoveUser(u) // ignore result
	return err
}

func handleUserLogout(w http.ResponseWriter) {
	w.WriteHeader(http.StatusSeeOther)
	w.Header().Set("Location", "/")
}

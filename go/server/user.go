package server

import (
	"fmt"
	"net/http"

	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
)

func (cfg Config) handleUserCreate(r *http.Request) error {
	err := r.ParseForm()
	if err != nil {
		return fmt.Errorf("parsing form: %w", err)
	}
	username := r.FormValue("username")
	password := r.FormValue("password_confirm")
	err = cfg.userDao.Create(db.NewUser(username, password))
	if err != nil {
		return err
	}
	return nil
}

func (cfg Config) handleUserLogin(r *http.Request) (db.User, error) {
	var u3 db.User
	err := r.ParseForm()
	if err != nil {
		return u3, fmt.Errorf("parsing form: %w", err)
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	u := db.NewUser(username, password)
	u2, err := cfg.userDao.Read(u)
	cfg.userDao.Read(db.NewUser(username, password))
	if err != nil {
		return u3, err
	}
	return u2, nil
}

func (cfg Config) handleUserUpdatePassword(r *http.Request) error {
	err := r.ParseForm()
	if err != nil {
		return fmt.Errorf("parsing form: %w", err)
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	newPassword := r.FormValue("password_confirm")
	u := db.NewUser(username, password)
	return cfg.userDao.UpdatePassword(u, newPassword)
}

func (cfg Config) handleUserDelete(r *http.Request) error {
	err := r.ParseForm()
	if err != nil {
		return fmt.Errorf("parsing form: %w", err)
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	u := db.NewUser(username, password)
	return cfg.userDao.Delete(u)
}

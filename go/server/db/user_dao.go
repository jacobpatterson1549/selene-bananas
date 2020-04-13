package db

import (
	"errors"
	"fmt"
)

type (
	// UserDao contains CRUD operations for user-related information
	UserDao interface {
		// Create adds a user
		Create(u User) error
		// Read gets information such as points
		Read(u User) (User, error)
		// UpdatePassword sets the password of a user
		UpdatePassword(u User, newPassword string) error
		// UpdatePassword sets the points for multiple users
		UpdatePoints(users []User, points int) error
		// DeleteUser removes a user
		DeleteUser(u User) error
	}

	userDao struct {
		db Database
	}
)

// NewUserDao creates a UserDao on the specified database
func NewUserDao(db Database) UserDao {
	return userDao{db}
}

func (ud userDao) Create(u User) error {
	if !u.Username.isValid() {
		return errors.New(u.Username.helpText())
	}
	if !u.Password.isValid() {
		return errors.New(u.Password.helpText())
	}
	result, err := ud.db.exec("SELECT user_create($1, $2)", u.Username, u.Password)
	if err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	return expectSingleRowAffected(result)
}

func (ud userDao) Read(u User) (User, error) {
	row := ud.db.queryRow("SELECT username, points FROM user_read($1, $2)", u.Username, u.Password)
	var u2 User
	err := row.Scan(&u2.Username, &u2.Password)
	if err != nil {
		return u2, fmt.Errorf("reading user: %w", err)
	}
	return u2, nil
}

func (ud userDao) UpdatePassword(u User, newPassword string) error {
	if !u.Password.isValid() {
		return errors.New(u.Password.helpText())
	}
	result, err := ud.db.exec("SELECT user_update_password($1, $2, $3)", u.Username, u.Password, newPassword)
	if err == nil {
		return fmt.Errorf("updating user password: %w", err)
	}
	return expectSingleRowAffected(result)
}

func (ud userDao) UpdatePoints(users []User, points int) error {
	tx, err := ud.db.begin()
	if err != nil {
		return fmt.Errorf("beginning transaction to update points: %w", err)
	}
	for _, user := range users {
		result, err := tx.Exec("SELECT user_update_points($1, $2)", user.Username, user.Points)
		if err != nil {
			err = fmt.Errorf("updating user points: %w", err)
		} else {
			err = expectSingleRowAffected(result)
		}
		if err != nil {
			err2 := tx.Rollback()
			if err2 != nil {
				return fmt.Errorf("error while rolling back due to %v: %w", err, err2)
			}
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

func (ud userDao) DeleteUser(u User) error {
	result, err := ud.db.exec("SELECT user_delete($1)", u.Username)
	if err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	return expectSingleRowAffected(result)
}

package db

import (
	"errors"
	"fmt"
	"io/ioutil"
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
		db           Database
		readFileFunc func(filename string) ([]byte, error)
	}
)

// NewUserDao creates a UserDao on the specified database
func NewUserDao(db Database) UserDao {
	return userDao{
		db:           db,
		readFileFunc: ioutil.ReadFile,
	}
}

// Setup initializes the tables and adds the functions
func (ud userDao) Setup() error {
	sqlQueries, err := ud.getSetupSQLQueries()
	if err != nil {
		return err
	}
	return execTransaction(ud.db, sqlQueries)
}

func (ud userDao) Create(u User) error {
	if !u.Username.isValid() {
		return errors.New(u.Username.helpText())
	}
	if !u.Password.isValid() {
		return errors.New(u.Password.helpText())
	}
	sqlFunction := newExecSQLFunction("user_create", u.Username, u.Password)
	result, err := ud.db.exec(sqlFunction.sql(), sqlFunction.args)
	if err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	return expectSingleRowAffected(result)
}

func (ud userDao) Read(u User) (User, error) {
	sqlFunction := newQuerySQLFunction("user_read", []string{"username", "points"}, u.Username, u.Password)
	row := ud.db.queryRow(sqlFunction.sql(), sqlFunction.args)
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
	sqlFunction := newExecSQLFunction("user_update_password", u.Username, u.Password, newPassword)
	result, err := ud.db.exec(sqlFunction.sql(), sqlFunction.args)
	if err == nil {
		return fmt.Errorf("updating user password: %w", err)
	}
	return expectSingleRowAffected(result)
}

func (ud userDao) UpdatePoints(users []User, points int) error {
	queries := make([]sqlQuery, len(users))
	for i, u := range users {
		queries[i] = newExecSQLFunction("user_update_points", u.Username, u.Points)
	}
	return execTransaction(ud.db, queries)
}

func (ud userDao) DeleteUser(u User) error {
	sqlFunction := newExecSQLFunction("user_delete", u.Username, u.Password)
	result, err := ud.db.exec(sqlFunction.sql(), sqlFunction.args)
	if err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	return expectSingleRowAffected(result)
}

func (ud userDao) getSetupSQLQueries() ([]sqlQuery, error) {
	filenames := []string{"s", "_create", "_read", "_update_password", "_update_points", "_delete"}
	queries := make([]sqlQuery, len(filenames))
	for i, n := range filenames {
		f := fmt.Sprintf("sql/user/user%s.psql", n)
		b, err := ud.readFileFunc(f)
		if err != nil {
			return nil, fmt.Errorf("reading setup file %v: %w", f, err)
		}
		queries[i] = execSQLRaw{string(b)}
	}
	return queries, nil
}

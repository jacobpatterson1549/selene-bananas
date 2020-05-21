package db

import (
	"fmt"
	"io/ioutil"
)

type (
	// UserDao contains CRUD operations for user-related information
	UserDao interface {
		// Setup initializes the tables and adds the functions
		Setup() error
		// Create adds a user
		Create(u User) error
		// Read gets information such as points
		Read(u User) (User, error)
		// UpdatePassword sets the password of a user
		UpdatePassword(u User, newPassword string) error
		// UpdatePassword increments the points for multiple users
		UpdatePointsIncrement(usernames []string, f UserPointsIncrementFunc) error
		// DeleteUser removes a user
		Delete(u User) error
	}

	// UserPointsIncrementFunc is used to determine how much to increment the points for a username
	UserPointsIncrementFunc func(username string) int

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

func (ud userDao) Setup() error {
	sqlQueries, err := ud.getSetupSQLQueries()
	if err != nil {
		return err
	}
	err = execTransaction(ud.db, sqlQueries)
	if err != nil {
		return fmt.Errorf("running setup query: %w", err)
	}
	return nil
}

func (ud userDao) Create(u User) error {
	hashedPassword, err := u.password.hash()
	if err != nil {
		return err
	}
	sqlFunction := newExecSQLFunction("user_create", u.Username, hashedPassword)
	result, err := ud.db.exec(sqlFunction.sql(), sqlFunction.args()...)
	if err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	err = sqlFunction.expectSingleRowAffected(result)
	if err != nil {
		return fmt.Errorf("user exists: %w", err)
	}
	return nil
}

func (ud userDao) Read(u User) (User, error) {
	sqlFunction := newQuerySQLFunction("user_read", []string{"username", "password", "points"}, u.Username)
	row := ud.db.queryRow(sqlFunction.sql(), sqlFunction.args()...)
	var u2, u3 User
	err := row.Scan(&u2.Username, &u2.password, &u2.Points)
	if err != nil {
		return u2, fmt.Errorf("reading user: %w", err)
	}
	hp := hashedPassword(u2.password)
	isCorrect, err := u.password.isCorrect(hp)
	switch {
	case err != nil:
		return u3, err
	case !isCorrect:
		return u3, fmt.Errorf("incorrect password")
	}
	return u2, nil
}

func (ud userDao) UpdatePassword(u User, newPassword string) error {
	p := password(newPassword)
	if !p.isValid() { // TODO: this is odd place to do validation.  Maybe other places are incorrect
		return fmt.Errorf(p.helpText())
	}
	hashedPassword, err := p.hash()
	if err != nil {
		return err
	}
	if _, err := ud.Read(u); err != nil { // check password
		return err
	}
	sqlFunction := newExecSQLFunction("user_update_password", u.Username, hashedPassword)
	result, err := ud.db.exec(sqlFunction.sql(), sqlFunction.args()...)
	if err != nil {
		return fmt.Errorf("updating user password: %w", err)
	}
	return sqlFunction.expectSingleRowAffected(result)
}

func (ud userDao) UpdatePointsIncrement(usernames []string, f UserPointsIncrementFunc) error {
	queries := make([]sqlQuery, len(usernames))
	for i, u := range usernames {
		pointsDelta := f(u)
		queries[i] = newExecSQLFunction("user_update_points_increment", u, pointsDelta)
	}
	return execTransaction(ud.db, queries)
}

func (ud userDao) Delete(u User) error {
	if _, err := ud.Read(u); err != nil { // check password
		return err
	}
	sqlFunction := newExecSQLFunction("user_delete", u.Username)
	result, err := ud.db.exec(sqlFunction.sql(), sqlFunction.args()...)
	if err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	return sqlFunction.expectSingleRowAffected(result)
}

func (ud userDao) getSetupSQLQueries() ([]sqlQuery, error) {
	filenames := []string{"s", "_create", "_read", "_update_password", "_update_points_increment", "_delete"}
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

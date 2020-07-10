package db

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"
)

type (
	// UserDao contains CRUD operations for user-related information
	UserDao struct {
		db           Database
		queryPeriod  time.Duration
		readFileFunc func(filename string) ([]byte, error)
	}

	// UserPointsIncrementFunc is used to determine how much to increment the points for a username
	UserPointsIncrementFunc func(username string) int

	// UserDaoConfig contains commonly shared UserDao properties
	UserDaoConfig struct {
		// Debug is a flag that causes the socket to log the types non-ping/pong messages that are read/written
		DB Database
		// QueryPeriod is the amount of time that any database action can take before it should timeout
		QueryPeriod time.Duration
	}
)

// NewUserDao creates a UserDao on the specified database
func (cfg UserDaoConfig) NewUserDao() (*UserDao, error) {
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("validating user dao creation: %w", err) // TODO: Change all validation error messages to be formatted like this.
	}
	ud := UserDao{
		db:           cfg.DB,
		queryPeriod:  cfg.QueryPeriod,
		readFileFunc: ioutil.ReadFile,
	}
	return &ud, nil
}

func (cfg UserDaoConfig) validate() error {
	switch {
	case cfg.DB == nil:
		return fmt.Errorf("database required")
	case cfg.QueryPeriod <= 0:
		return fmt.Errorf("positive query period required")
	}
	return nil
}

// Setup initializes the tables and adds the functions
func (ud UserDao) Setup(ctx context.Context) error {
	ctx, cancelFunc := context.WithTimeout(ctx, ud.queryPeriod)
	defer cancelFunc()
	queries, err := ud.setupSQLQueries()
	if err != nil {
		return fmt.Errorf("creating setup query: %w", err)
	}
	if err = ud.db.exec(ctx, queries...); err != nil {
		return fmt.Errorf("running setup query: %w", err)
	}
	return nil
}

// Create adds a user
func (ud UserDao) Create(ctx context.Context, u User) error {
	ctx, cancelFunc := context.WithTimeout(ctx, ud.queryPeriod)
	defer cancelFunc()
	hashedPassword, err := u.hashPassword()
	if err != nil {
		return err
	}
	q := newExecSQLFunction("user_create", u.Username, hashedPassword)
	if err := ud.db.exec(ctx, q); err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	return nil
}

// Read gets information such as points
func (ud UserDao) Read(ctx context.Context, u User) (User, error) {
	ctx, cancelFunc := context.WithTimeout(ctx, ud.queryPeriod)
	defer cancelFunc()
	q := newQuerySQLFunction("user_read", []string{"username", "password", "points"}, u.Username)
	row := ud.db.queryRow(ctx, q)
	var u2 User
	if err := row.Scan(&u2.Username, &u2.password, &u2.Points); err != nil {
		return User{}, fmt.Errorf("reading user: %w", err)
	}
	hashedPassword := []byte(u2.password)
	isCorrect, err := u.isCorrectPassword(hashedPassword)
	switch {
	case err != nil:
		return User{}, fmt.Errorf("reading user: %w", err)
	case !isCorrect:
		return User{}, fmt.Errorf("incorrect password")
	}
	return u2, nil
}

// UpdatePassword sets the password of a user
func (ud UserDao) UpdatePassword(ctx context.Context, u User, newP string) error {
	ctx, cancelFunc := context.WithTimeout(ctx, ud.queryPeriod)
	defer cancelFunc()
	if err := validatePassword(newP); err != nil {
		return err
	}
	u.password = newP
	hashedPassword, err := u.hashPassword()
	if err != nil {
		return err
	}
	if _, err := ud.Read(ctx, u); err != nil {
		return fmt.Errorf("checking password: %w", err)
	}
	q := newExecSQLFunction("user_update_password", u.Username, hashedPassword)
	if err := ud.db.exec(ctx, q); err != nil {
		return fmt.Errorf("updating user password: %w", err)
	}
	return nil
}

// UpdatePointsIncrement increments the points for multiple users
func (ud UserDao) UpdatePointsIncrement(ctx context.Context, usernames []string, f UserPointsIncrementFunc) error {
	ctx, cancelFunc := context.WithTimeout(ctx, ud.queryPeriod)
	defer cancelFunc()
	queries := make([]sqlQuery, len(usernames))
	for i, u := range usernames {
		pointsDelta := f(u)
		queries[i] = newExecSQLFunction("user_update_points_increment", u, pointsDelta)
	}
	if err := ud.db.exec(ctx, queries...); err != nil {
		return fmt.Errorf("incrementing user points: %w", err)
	}
	return nil
}

// Delete removes a user
func (ud UserDao) Delete(ctx context.Context, u User) error {
	ctx, cancelFunc := context.WithTimeout(ctx, ud.queryPeriod)
	defer cancelFunc()
	if _, err := ud.Read(ctx, u); err != nil { // check password
		return err
	}
	q := newExecSQLFunction("user_delete", u.Username)
	if err := ud.db.exec(ctx, q); err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	return nil
}

func (ud UserDao) setupSQLQueries() ([]sqlQuery, error) {
	filenames := []string{"s", "_create", "_read", "_update_password", "_update_points_increment", "_delete"}
	queries := make([]sqlQuery, len(filenames))
	for i, n := range filenames {
		f := fmt.Sprintf("resources/sql/user%s.sql", n)
		b, err := ud.readFileFunc(f)
		if err != nil {
			return nil, fmt.Errorf("reading setup file %v: %w", f, err)
		}
		queries[i] = execSQLRaw{string(b)}
	}
	return queries, nil
}

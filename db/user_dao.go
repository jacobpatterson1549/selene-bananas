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
		return nil, err
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
	sqlQueries, err := ud.setupSQLQueries()
	if err != nil {
		return err
	}
	err = execTransaction(ctx, ud.db, sqlQueries)
	if err != nil {
		return fmt.Errorf("running setup query: %w", err)
	}
	return nil
}

// Create adds a user
func (ud UserDao) Create(ctx context.Context, u User) error {
	ctx, cancelFunc := context.WithTimeout(ctx, ud.queryPeriod)
	defer cancelFunc()
	hashedPassword, err := u.password.hash()
	if err != nil {
		return err
	}
	sqlFunction := newExecSQLFunction("user_create", u.Username, hashedPassword)
	result, err := ud.db.exec(ctx, sqlFunction.sql(), sqlFunction.args()...)
	if err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	err = sqlFunction.expectSingleRowAffected(result)
	if err != nil {
		return fmt.Errorf("user exists: %w", err)
	}
	return nil
}

// Read gets information such as points
func (ud UserDao) Read(ctx context.Context, u User) (User, error) {
	ctx, cancelFunc := context.WithTimeout(ctx, ud.queryPeriod)
	defer cancelFunc()
	sqlFunction := newQuerySQLFunction("user_read", []string{"username", "password", "points"}, u.Username)
	row := ud.db.queryRow(ctx, sqlFunction.sql(), sqlFunction.args()...)
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

// UpdatePassword sets the password of a user
func (ud UserDao) UpdatePassword(ctx context.Context, u User, newP string) error {
	ctx, cancelFunc := context.WithTimeout(ctx, ud.queryPeriod)
	defer cancelFunc()
	p, err := newPassword(newP)
	if err != nil {
		return err
	}
	hashedPassword, err := p.hash()
	if err != nil {
		return err
	}
	if _, err := ud.Read(ctx, u); err != nil { // check password
		return err
	}
	sqlFunction := newExecSQLFunction("user_update_password", u.Username, hashedPassword)
	result, err := ud.db.exec(ctx, sqlFunction.sql(), sqlFunction.args()...)
	if err != nil {
		return fmt.Errorf("updating user password: %w", err)
	}
	return sqlFunction.expectSingleRowAffected(result)
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
	return execTransaction(ctx, ud.db, queries)
}

// Delete removes a user
func (ud UserDao) Delete(ctx context.Context, u User) error {
	ctx, cancelFunc := context.WithTimeout(ctx, ud.queryPeriod)
	defer cancelFunc()
	if _, err := ud.Read(ctx, u); err != nil { // check password
		return err
	}
	sqlFunction := newExecSQLFunction("user_delete", u.Username)
	result, err := ud.db.exec(ctx, sqlFunction.sql(), sqlFunction.args()...)
	if err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	return sqlFunction.expectSingleRowAffected(result)
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

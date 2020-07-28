package db

import (
	"context"
	"fmt"
)

type (
	// UserDao contains CRUD operations for user-related information.
	UserDao struct {
		db           Database
		readFileFunc func(filename string) ([]byte, error)
	}

	// UserDaoConfig contains commonly shared UserDao properties.
	UserDaoConfig struct {
		// Debug is a flag that causes the socket to log the types non-ping/pong messages that are read/written.
		DB Database
		// ReadFileFunc is used to fetch setup queries.
		ReadFileFunc func(filename string) ([]byte, error)
	}
)

// NewUserDao creates a UserDao on the specified database.
func (cfg UserDaoConfig) NewUserDao() (*UserDao, error) {
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("creating user dao: validation: %w", err)
	}
	ud := UserDao{
		db:           cfg.DB,
		readFileFunc: cfg.ReadFileFunc,
	}
	return &ud, nil
}

func (cfg UserDaoConfig) validate() error {
	switch {
	case cfg.DB == nil:
		return fmt.Errorf("database required")
	case cfg.ReadFileFunc == nil:
		return fmt.Errorf("read file func required")
	}
	if _, ok := cfg.DB.(sqlDatabase); !ok {
		return fmt.Errorf("only sql database is supported")
	}
	return nil
}

// Setup initializes the tables and adds the functions.
func (ud UserDao) Setup(ctx context.Context) error {
	queries, err := ud.setupSQLQueries()
	if err != nil {
		return fmt.Errorf("creating setup query: %w", err)
	}
	if err = ud.db.exec(ctx, queries...); err != nil {
		return fmt.Errorf("running setup query: %w", err)
	}
	return nil
}

// Create adds a user.
func (ud UserDao) Create(ctx context.Context, u User) error {
	hashedPassword, err := u.hashPassword()
	if err != nil {
		return err
	}
	q := newSQLExecFunction("user_create", u.Username, hashedPassword)
	if err := ud.db.exec(ctx, q); err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	return nil
}

// Read gets information such as points.
func (ud UserDao) Read(ctx context.Context, u User) (*User, error) {
	cols := []string{
		"username",
		"password",
		"points",
	}
	q := newSQLQueryFunction("user_read", cols, u.Username)
	row := ud.db.query(ctx, q)
	var u2 User
	if err := row.Scan(&u2.Username, &u2.password, &u2.Points); err != nil {
		return nil, fmt.Errorf("reading user: %w", err)
	}
	hashedPassword := []byte(u2.password)
	isCorrect, err := u.isCorrectPassword(hashedPassword)
	switch {
	case err != nil:
		return nil, fmt.Errorf("reading user: %w", err)
	case !isCorrect:
		return nil, fmt.Errorf("incorrect password")
	}
	return &u2, nil
}

// UpdatePassword sets the password of a user.
func (ud UserDao) UpdatePassword(ctx context.Context, u User, newP string) error {
	if err := validatePassword(newP); err != nil {
		return err
	}
	hashedPassword, err := u.hashPassword()
	if err != nil {
		return err
	}
	if _, err := ud.Read(ctx, u); err != nil {
		return fmt.Errorf("checking password: %w", err)
	}
	q := newSQLExecFunction("user_update_password", u.Username, hashedPassword)
	if err := ud.db.exec(ctx, q); err != nil {
		return fmt.Errorf("updating user password: %w", err)
	}
	return nil
}

// UpdatePointsIncrement increments the points for multiple users by the amount defined in the map.
func (ud UserDao) UpdatePointsIncrement(ctx context.Context, userPoints map[string]int) error {
	queries := make([]query, 0, len(userPoints))
	for username, points := range userPoints {
		queries = append(queries, newSQLExecFunction("user_update_points_increment", username, points))
	}
	if err := ud.db.exec(ctx, queries...); err != nil {
		return fmt.Errorf("incrementing user points: %w", err)
	}
	return nil
}

// Delete removes a user.
func (ud UserDao) Delete(ctx context.Context, u User) error {
	if _, err := ud.Read(ctx, u); err != nil {
		return fmt.Errorf("checking password: %w", err)
	}
	q := newSQLExecFunction("user_delete", u.Username)
	if err := ud.db.exec(ctx, q); err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	return nil
}

func (ud UserDao) setupSQLQueries() ([]query, error) {
	filenames := []string{
		"s",
		"_create",
		"_read",
		"_update_password",
		"_update_points_increment",
		"_delete",
	}
	queries := make([]query, len(filenames))
	for i, n := range filenames {
		f := fmt.Sprintf("resources/sql/user%s.sql", n)
		b, err := ud.readFileFunc(f)
		if err != nil {
			return nil, fmt.Errorf("reading setup file %v: %w", f, err)
		}
		q := sqlExecRaw(string(b))
		queries[i] = q
	}
	return queries, nil
}

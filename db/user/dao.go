package user

import (
	"context"
	"fmt"

	"github.com/jacobpatterson1549/selene-bananas/db"
	"github.com/jacobpatterson1549/selene-bananas/db/sql"
)

type (
	// Dao contains CRUD operations for user-related information.
	Dao struct {
		db           db.Database
		readFileFunc func(filename string) ([]byte, error)
	}

	// DaoConfig contains commonly shared Dao properties.
	DaoConfig struct {
		// Debug is a flag that causes the socket to log the types non-ping/pong messages that are read/written.
		DB db.Database
		// ReadFileFunc is used to fetch setup queries.
		ReadFileFunc func(filename string) ([]byte, error)
	}
)

// NewDao creates a Dao on the specified database.
func (cfg DaoConfig) NewDao() (*Dao, error) {
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("creating user dao: validation: %w", err)
	}
	d := Dao{
		db:           cfg.DB,
		readFileFunc: cfg.ReadFileFunc,
	}
	return &d, nil
}

func (cfg DaoConfig) validate() error {
	switch {
	case cfg.DB == nil:
		return fmt.Errorf("database required")
	case cfg.ReadFileFunc == nil:
		return fmt.Errorf("read file func required")
	}
	if _, ok := cfg.DB.(sql.Database); !ok {
		return fmt.Errorf("only sql database is supported")
	}
	return nil
}

// Setup initializes the tables and adds the functions.
func (d Dao) Setup(ctx context.Context) error {
	queries, err := d.SetupQueries()
	if err != nil {
		return fmt.Errorf("creating setup query: %w", err)
	}
	if err = d.db.Exec(ctx, queries...); err != nil {
		return fmt.Errorf("running setup query: %w", err)
	}
	return nil
}

// Create adds a user.
func (d Dao) Create(ctx context.Context, u User) error {
	hashedPassword, err := u.hashPassword()
	if err != nil {
		return err
	}
	q := sql.NewExecFunction("user_create", u.Username, hashedPassword)
	if err := d.db.Exec(ctx, q); err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	return nil
}

// Read gets information such as points.
func (d Dao) Read(ctx context.Context, u User) (*User, error) {
	cols := []string{
		"username",
		"password",
		"points",
	}
	q := sql.NewQueryFunction("user_read", cols, u.Username)
	row := d.db.Query(ctx, q)
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
func (d Dao) UpdatePassword(ctx context.Context, u User, newP string) error {
	if _, err := d.Read(ctx, u); err != nil {
		return fmt.Errorf("checking password: %w", err)
	}
	if err := validatePassword(newP); err != nil {
		return err
	}
	u.password = newP
	hashedPassword, err := u.hashPassword()
	if err != nil {
		return err
	}
	q := sql.NewExecFunction("user_update_password", u.Username, hashedPassword)
	if err := d.db.Exec(ctx, q); err != nil {
		return fmt.Errorf("updating user password: %w", err)
	}
	return nil
}

// UpdatePointsIncrement increments the points for multiple users by the amount defined in the map.
func (d Dao) UpdatePointsIncrement(ctx context.Context, userPoints map[string]int) error {
	queries := make([]db.Query, 0, len(userPoints))
	for username, points := range userPoints {
		queries = append(queries, sql.NewExecFunction("user_update_points_increment", username, points))
	}
	if err := d.db.Exec(ctx, queries...); err != nil {
		return fmt.Errorf("incrementing user points: %w", err)
	}
	return nil
}

// Delete removes a user.
func (d Dao) Delete(ctx context.Context, u User) error {
	if _, err := d.Read(ctx, u); err != nil {
		return fmt.Errorf("checking password: %w", err)
	}
	q := sql.NewExecFunction("user_delete", u.Username)
	if err := d.db.Exec(ctx, q); err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	return nil
}

// SetupQueries gets the queries to setup backing tables and functions.
func (d Dao) SetupQueries() ([]db.Query, error) {
	filenames := []string{
		"s",
		"_create",
		"_read",
		"_update_password",
		"_update_points_increment",
		"_delete",
	}
	queries := make([]db.Query, len(filenames))
	for i, n := range filenames {
		f := fmt.Sprintf("resources/sql/user%s.sql", n)
		b, err := d.readFileFunc(f)
		if err != nil {
			return nil, fmt.Errorf("reading setup file %v: %w", f, err)
		}
		q := sql.RawQuery(b)
		queries[i] = q
	}
	return queries, nil
}

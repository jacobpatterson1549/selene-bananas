// Package db interacts with storing user attributes so they can be retrieved after the server restarts
package db

import (
	"context"
	"io"
)

type (
	// Database contains methods to create, read, update, and delete data.
	Database interface {
		// Setup initializes the database by reading the files.
		Setup(ctx context.Context, files []io.Reader) error
		// Query reads from the database without updating it.
		Query(ctx context.Context, q Query) Scanner
		// Exec makes a change to existing data, creating/modifying/removing it.
		Exec(ctx context.Context, queries ...Query) error
	}

	// Scanner reads data from the database.
	Scanner interface {
		// Scan reads from the database into the destination array.
		Scan(dest ...interface{}) error
	}

	// Query is a message that is sent to the database.
	Query interface {
		// cmd is the injection-safe message to send to the database.
		Cmd() string
		// args are the user-provided properties of the messages which shuld be escaped.
		Args() []interface{}
	}
)

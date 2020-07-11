// Package db interacts with storing user attributes so they can be retrieved after the server restarts
package db

import (
	"context"
)

type (
	// Database contains methods to create, read, update, and delete data
	Database interface {
		// query reads from the database without updating it
		query(ctx context.Context, q query) scanner
		// exec makes a change to existing data, creating/modifying/removing it
		exec(ctx context.Context, queries ...query) error
	}

	scanner interface {
		// Scan reads from the database into the destination array.
		Scan(dest ...interface{}) error
	}

	// query is a message that is sent to the database
	query interface {
		// cmd is the injection-safe message to send to the database.
		cmd() string
		// args are the user-provided properties of the messages which shuld be escaped
		args() []interface{}
	}
)

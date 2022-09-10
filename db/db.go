// Package db provides permanent data storage
package db

import "time"

// Config contains options for how the database should run.
type Config struct {
	// QueryPeriod is the amount of time that any database action can take before it should timeout.
	QueryPeriod time.Duration
}

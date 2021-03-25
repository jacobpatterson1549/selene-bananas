// Package log provides an abstraction over log.Logger.
package log

// Logger is an interface over log.Logger to ensure the same log is used in most places rather than the default logger in that package.
type Logger interface {
	// Printf calls writes the formatted string with values to the logger.
	// Arguments are handled in the manner of fmt.Printf.
	Printf(format string, v ...interface{})
}

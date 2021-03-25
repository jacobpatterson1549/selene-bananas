// Package logtest implements support for testing Loggers.
package logtest

import (
	"bytes"
	"fmt"

	"github.com/jacobpatterson1549/selene-bananas/server/log"
)

// DiscardLogger is a Logger that writes anything and everything to io.Discard.
var DiscardLogger = &discardLogger{}

// NewLogger creates a Logger.
func NewLogger() *Logger {
	var buf bytes.Buffer
	l := Logger{
		buf: &buf,
	}
	return &l
}

// discardLogger is a logger that logs nothing.
// This is more simple than using the standard log.Logger:New() with the io.Discard writer.
type discardLogger struct{}

// DiscardLogger (and other log.Loggers) implement the server's log.Logger interface.
var _ log.Logger = DiscardLogger

// Printf implements the log.Logger interface
func (discardLogger) Printf(format string, v ...interface{}) {
	// NOOP
}

// Logger is a logger that writes to a buffer to be read later.
type Logger struct {
	buf *bytes.Buffer
}

// Logger implements the server's log.Logger interface.
var _ log.Logger = NewLogger()

// Printf implements the log.Logger interface
func (l *Logger) Printf(format string, v ...interface{}) {
	fmt.Fprintf(l.buf, format, v...)
}

// String returns the recorded string.
func (l *Logger) String() string {
	return l.buf.String()
}

// Empty returns if buffer is empty.
func (l *Logger) Empty() bool {
	return l.buf.Len() == 0
}

// Reset resets the buffer to be empty,
func (l *Logger) Reset() {
	l.buf.Reset()
}

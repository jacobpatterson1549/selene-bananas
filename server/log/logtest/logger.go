// Package logtest implements support for testing Loggers.
package logtest

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/jacobpatterson1549/selene-bananas/server/log"
)

// DiscardLogger is a Logger that writes anything and everything to io.Discard.
var DiscardLogger = new(discardLogger)

// NewLogger creates a Logger.
func NewLogger() *Logger {
	l := Logger{
		buf: new(bytes.Buffer),
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
	mu  sync.RWMutex
}

// Logger implements the server's log.Logger interface.
var _ log.Logger = NewLogger()

// Printf implements the log.Logger interface
func (l *Logger) Printf(format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.buf, format, v...)
}

// String returns the recorded string.
func (l *Logger) String() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.buf.String()
}

// Empty returns if buffer is empty.
func (l *Logger) Empty() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Len() == 0
}

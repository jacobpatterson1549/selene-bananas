// Package runner is used to run parts of the server once and only once.
package runner

import (
	"fmt"
	"sync"
)

// Runner is a thread-safe structure that can be run, finished, and queried.
type Runner struct {
	runMu   sync.Mutex
	running bool
	runDone bool
}

// Run starts the running the runner.  If it already running, it returns an error
func (r *Runner) Run() error {
	r.runMu.Lock()
	defer r.runMu.Unlock()
	if r.running || r.runDone {
		return fmt.Errorf("already running or has finished running, it can only be run once")
	}
	r.running = true
	return nil
}

// Finish marks the runner as done, regardless if it ran.
func (r *Runner) Finish() {
	r.runMu.Lock()
	defer r.runMu.Unlock()
	r.running = false
	r.runDone = true
}

// IsRunning determines if the runner is running
func (r *Runner) IsRunning() bool {
	r.runMu.Lock()
	defer r.runMu.Unlock()
	return r.running
}

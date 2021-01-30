package runner_test

import (
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/server/runner"
)

func TestRun(t *testing.T) {
	var r runner.Runner
	err1 := r.Run()
	if err1 != nil {
		t.Errorf("unwanted error running: %v", err1)
	}
	err2 := r.Run()
	if err2 == nil {
		t.Error("wanted error running while it is running")
	}
	r.Finish()
	err3 := r.Run()
	if err3 == nil {
		t.Error("wanted error running after it is done running")
	}
}

func TestIsRunning(t *testing.T) {
	var r runner.Runner
	isRunning1 := r.IsRunning()
	if isRunning1 {
		t.Error("did not want runner be running before it is run")
	}
	err := r.Run()
	if err != nil {
		t.Errorf("unwanted error running: %v", err)
	}
	isRunning2 := r.IsRunning()
	if !isRunning2 {
		t.Error("wanted runner to be running while it is running")
	}
	r.Finish()
	isRunning3 := r.IsRunning()
	if isRunning3 {
		t.Error("did not want runner be running after it is run")
	}
}

func BenchmarkRun(b *testing.B) {
	n := b.N
	startRun := make(chan struct{})
	calcDone := make(chan struct{})
	runStarted := make(chan struct{}, n)
	runFailed := make(chan struct{}, n)
	var trigger struct{}
	var r runner.Runner
	numRunStarted := 0
	numRunFailed := 0
	go func() {
		defer close(calcDone)
		for {
			select {
			case <-runStarted:
				numRunStarted++
			case <-runFailed:
				numRunFailed++
			}
			if numRunStarted+numRunFailed == n {
				return
			}
		}
	}()
	for i := 0; i < n; i++ {
		go func() {
			<-startRun
			err := r.Run()
			if err == nil {
				runStarted <- trigger
			} else {
				runFailed <- trigger
			}
		}()
	}
	close(startRun) // start the goroutines
	<-calcDone
	wantNumRunFailed := n - 1
	switch {
	case numRunStarted != 1:
		b.Errorf("wanted the run to only be started once, got %v", numRunStarted)
	case wantNumRunFailed != numRunFailed:
		b.Errorf("wanted the run to fail to start %v times because it only started once, got %v", wantNumRunFailed, numRunFailed)
	}
}

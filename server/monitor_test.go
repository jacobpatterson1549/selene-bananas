package server

import (
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestWriteMemoryStats(t *testing.T) {
	m := runtime.MemStats{
		Alloc:      9,
		TotalAlloc: 8,
		Sys:        7,
		Mallocs:    6,
		Frees:      5,
	}
	var w strings.Builder
	writeMemoryStats(&w, &m)
	got := w.String()
	for i, v := range []string{"9", "8", "7", "1"} {
		want := " " + v + "\n" // no negatives or trailing digits
		if !strings.Contains(got, want) {
			t.Errorf("Test %v: wanted memory stats output to contain %v:\n%v", i, want, got)
		}
	}
}

func TestWriteGoroutineExpectations(t *testing.T) {
	var numExpectations [2]int
	for i, hasTLS := range []bool{true, false} {
		var w strings.Builder
		writeGoroutineExpectations(&w, hasTLS)
		expectations := w.String()
		lines := strings.Split(expectations, "\n")
		for _, e := range lines {
			if strings.HasPrefix(e, "* ") {
				numExpectations[i]++
			}
		}
		want := strconv.Itoa(numExpectations[i])
		if len(lines) < 2 || !strings.Contains(lines[1], want) {
			t.Errorf("server %v: wanted %v goroutine expectations", i, want)
		}
	}
	if numExpectations[0] == numExpectations[1] {
		t.Error("wanted different goroutine expectations for http-only and http/https server")
	}
}

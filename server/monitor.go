package server

import (
	"fmt"
	"io"
	"net/http"
	"runtime"
	"runtime/pprof"
)

// handleMonitor writes runtime information to the response.
func (s Server) handleMonitor(w http.ResponseWriter, r *http.Request) {
	m := new(runtime.MemStats)
	runtime.ReadMemStats(m)
	hasTLS := s.validHTTPAddr()
	p := pprof.Lookup("goroutine")
	writeMemoryStats(w, m)
	writeLn(w)
	writeGoroutineExpectations(w, hasTLS)
	writeLn(w)
	writeGoroutineStackTraces(w, p)
}

// writeMemoryStats writes the memory runtime statistics of the server.
func writeMemoryStats(w io.Writer, m *runtime.MemStats) {
	writeLn(w, "--- Memory Stats ---")
	writeLn(w, "Alloc (bytes on heap)", m.Alloc)
	writeLn(w, "TotalAlloc (total heap size)", m.TotalAlloc)
	writeLn(w, "Sys (bytes used to run server)", m.Sys)
	writeLn(w, "Live object count (Mallocs - Frees)", m.Mallocs-m.Frees)
}

// writeGoroutineExpectations writes a message about the expected goroutines.
func writeGoroutineExpectations(w io.Writer, hasTLS bool) {
	writeLn(w, "--- Goroutine Expectations ---")
	switch {
	case hasTLS:
		writeLn(w, "Ten (10) goroutines are expected on an idling server.")
		writeLn(w, "Note that the first two goroutines create extra threads for each tls connection.")
		writeLn(w, "* a goroutine listening for interrupt/termination signals so the server can stop gracefully")
		writeLn(w, "* a goroutine to handle tls connections")
		writeLn(w, "* a goroutine to run the https (tls) server")
	default:
		writeLn(w, "Seven (7) goroutines are expected on an idling server.")
	}
	writeLn(w, "* a goroutine to run the http server")
	writeLn(w, "* a goroutine to open new sql database connections")
	writeLn(w, "* a goroutine to reset existing sql database connections")
	writeLn(w, "* a goroutine to serve http/2 requests")
	writeLn(w, "* a goroutine to run the lobby")
	writeLn(w, "* a goroutine to run the main procedure")
	writeLn(w, "* a goroutine to write profiling information about goroutines")
	writeLn(w, "Each player in the lobby should have two (2) goroutines to read and write websocket messages.")
	writeLn(w, "Each game in the lobby runs on a single (1) goroutine.")
}

// writeGoroutineStackTraces writes the goroutine runitme profile's stack traces.
func writeGoroutineStackTraces(w io.Writer, p *pprof.Profile) {
	writeLn(w, "--- Goroutine Stack Traces ---")
	p.WriteTo(w, 1)
}

// writeLn writes the interfaces, followed by a newline, to the writer.
func writeLn(w io.Writer, a ...interface{}) {
	w.Write([]byte(fmt.Sprintln(a...)))
}

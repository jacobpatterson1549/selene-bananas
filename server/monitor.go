package server

import (
	"fmt"
	"io"
	"net/http"
	"runtime"
	"runtime/pprof"
)

// handleMonitor writes runtime information to the response.
func (s *Server) handleMonitor(w http.ResponseWriter, r *http.Request) {
	m := new(runtime.MemStats)
	runtime.ReadMemStats(m)
	hasTLS := s.validHTTPAddr()
	p := pprof.Lookup("goroutine")
	writeMemoryStats(w, m)
	fmt.Fprintln(w)
	writeGoroutineExpectations(w, hasTLS)
	fmt.Fprintln(w)
	writeGoroutineStackTraces(w, p)
}

// writeMemoryStats writes the memory runtime statistics of the server.
func writeMemoryStats(w io.Writer, m *runtime.MemStats) {
	fmt.Fprintln(w, "--- Memory Stats ---")
	fmt.Fprintln(w, "Alloc (bytes on heap)", m.Alloc)
	fmt.Fprintln(w, "TotalAlloc (total heap size)", m.TotalAlloc)
	fmt.Fprintln(w, "Sys (bytes used to run server)", m.Sys)
	fmt.Fprintln(w, "Live object count (Mallocs - Frees)", m.Mallocs-m.Frees)
}

// writeGoroutineExpectations writes a message about the expected goroutines.
func writeGoroutineExpectations(w io.Writer, hasTLS bool) {
	fmt.Fprintln(w, "--- Goroutine Expectations ---")
	switch {
	case hasTLS:
		fmt.Fprintln(w, "Ten (10) goroutines are expected on an idling server.")
		fmt.Fprintln(w, "Note that the first two goroutines create extra threads for each tls connection.")
		fmt.Fprintln(w, "* a goroutine listening for interrupt/termination signals so the server can stop gracefully")
		fmt.Fprintln(w, "* a goroutine to handle tls connections")
		fmt.Fprintln(w, "* a goroutine to run the https (tls) server")
	default:
		fmt.Fprintln(w, "Seven (7) goroutines are expected on an idling server.")
	}
	fmt.Fprintln(w, "* a goroutine to run the http server")
	fmt.Fprintln(w, "* a goroutine to open new sql database connections")
	fmt.Fprintln(w, "* a goroutine to reset existing sql database connections")
	fmt.Fprintln(w, "* a goroutine to serve http/2 requests")
	fmt.Fprintln(w, "* a goroutine to run the lobby")
	fmt.Fprintln(w, "* a goroutine to run the main procedure")
	fmt.Fprintln(w, "* a goroutine to write profiling information about goroutines")
	fmt.Fprintln(w, "Each player in the lobby should have two (2) goroutines to read and write websocket messages.")
	fmt.Fprintln(w, "Each game in the lobby runs on a single (1) goroutine.")
}

// writeGoroutineStackTraces writes the goroutine runitme profile's stack traces.
func writeGoroutineStackTraces(w io.Writer, p *pprof.Profile) {
	fmt.Fprintln(w, "--- Goroutine Stack Traces ---")
	p.WriteTo(w, 1)
}

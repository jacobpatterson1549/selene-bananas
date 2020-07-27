package server

import (
	"fmt"
	"net/http"
	"runtime"
	"runtime/pprof"
)

// handleMonitor writes runtime information to the response.
func (s Server) handleMonitor(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	goroutineProfiles := pprof.Lookup("goroutine")
	lines := [][]interface{}{
		{"--- Memory Stats ---"},
		{"Alloc (bytes on heap)", m.Alloc},
		{"TotalAlloc (total heap size)", m.TotalAlloc},
		{"Sys (bytes used to run server)", m.Sys},
		{"Live object count (Mallocs - Frees)", m.Mallocs - m.Frees},
	}
	for _, e := range s.goroutineExpectations() {
		lines = append(lines, []interface{}{e})
	}
	lines = append(lines, []interface{}{"--- goroutine stack traces ---"})
	for _, l := range lines {
		w.Write([]byte(fmt.Sprintln(l...)))
	}
	goroutineProfiles.WriteTo(w, 1)
}

// goroutineExpectations returns a message about the expected goroutines.
func (s Server) goroutineExpectations() []string {
	var e []string
	e = append(e, "")
	switch {
	case s.validHTTPAddr():
		e = append(e, "10 goroutines are expected on an idling server.")
		e = append(e, "Note that the first two goroutines create extra threads for each tls connection")
		e = append(e, "* a goroutine listening for interrupt/termination signals so the server can stop gracefully")
		e = append(e, "* a goroutine to handle tls connections")
		e = append(e, "* a goroutine to run the https (tls) server")
	default:
		e = append(e, "7 goroutines are expected on an idling server.")
	}
	e = append(e, "* a goroutine to run the http server")
	e = append(e, "* a goroutine to open new sql database connections")
	e = append(e, "* a goroutine to reset existing sql database connections")
	e = append(e, "* a goroutine to serve http/2 requests")
	e = append(e, "* a goroutine to run the lobby")
	e = append(e, "* a goroutine to run the main procedure")
	e = append(e, "* a goroutine to write profiling information about goroutines")
	e = append(e, "")
	e = append(e, "Each player in the lobby should have two (2) goroutines to read and write websocket messages.")
	e = append(e, "Each game in the lobby runs on a single (1) goroutine.")
	e = append(e, "")
	return e
}

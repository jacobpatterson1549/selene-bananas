package server

import (
	"fmt"
	"net/http"
	"runtime"
	"runtime/pprof"
)

const goroutineExpectations = `10-12 goroutines are expected on an idling server.
If any user has logged into the server, the first two goroutines will report "2" above the stack trace instead of "1," resulting in 12 total goroutines if no users are in the lobby.
* a goroutine listening for interrut/termination signals so the server can stop gracefully
* a goroutine to handle tls connections
* a goroutine to run the https (tls) server
* a goroutine to run the http server
* a goroutine to open new sql database connections
* a goroutine to reset existing sql database connections
* a goroutine to serve http/2 requests
* a goroutine to run the lobby
* a goroutine to run the main procedure
* a goroutine to write profiling information about goroutines

Each player in the lobby should have two (2) goroutines to read and write websocket messages.
Each game in the lobby runs on a single (1) goroutine.
`

// handleMonitor writes runtime information to the response
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
		{},
		{goroutineExpectations},
		{"--- goroutine stack traces ---"},
	}
	for _, l := range lines {
		w.Write([]byte(fmt.Sprintln(l...)))
	}
	goroutineProfiles.WriteTo(w, 1)
}

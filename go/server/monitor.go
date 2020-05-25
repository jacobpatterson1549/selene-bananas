package server

import (
	"fmt"
	"net/http"
	"runtime"
	"runtime/pprof"
)

func (s Server) handleMonitor(w http.ResponseWriter, r *http.Request) error {
	write := func(v ...interface{}) {
		w.Write([]byte(fmt.Sprintln(v...)))
	}

	w.Write([]byte("--- Memory Stats ---\n"))
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	write("Alloc (bytes on heap)", m.Alloc)
	write("TotalAlloc (total heap size)", m.TotalAlloc)
	write("Sys (bytes used to run server)", m.Sys)
	write("Live object count (Mallocs - Frees)", m.Mallocs-m.Frees)

	write()

	goroutineExpectations := `Eight (8) goroutines are expected on an idling server:
* a goroutine listening for interrut/termination signals so the server can stop gracefully
* a goroutine to run the server
* a goroutine to listen for new server requests
* a goroutine to run the main procedure
* a goroutine to keep open new sql database connections
* a goroutine to reset existing sql database connections
* a goroutine to run the lobby
* a goroutine to write profiling information about goroutines

Each player in the lobby should have two (2) goroutines to read and write websocket messages.
Each game in the lobby runs on a single (1) goroutine.
`
	write(goroutineExpectations)
	write("--- goroutine stack traces ---")
	goroutineProfiles := pprof.Lookup("goroutine")
	goroutineProfiles.WriteTo(w, 1)

	return nil
}

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

	write("--- goroutine stack traces ---")
	goroutineProfiles := pprof.Lookup("goroutine")
	goroutineProfiles.WriteTo(w, 1)

	return nil
}

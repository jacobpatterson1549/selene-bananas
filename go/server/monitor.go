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
	write("numGC", m.NumGC)
	// write("garbage collector run times [newest-oldest]", m.PauseNs)
	write("garbage collector run times [newest-oldest]", pauseNs(m))

	write()

	write("--- goroutine stack traces ---")
	goroutineProfiles := pprof.Lookup("goroutine")
	goroutineProfiles.WriteTo(w, 1)

	return nil
}

// pauseNs gets the recent garbage collection pause times, ordered from newest to oldest
func pauseNs(m runtime.MemStats) [256]uint64 {
	circularPauseNs := m.PauseNs
	var i uint32
	switch {
	case m.NumGC <= 256:
		i = 0
	default:
		i = m.NumGC % 256
	}
	var linearPauseNs [256]uint64
	copy(linearPauseNs[0:], circularPauseNs[i:])
	copy(linearPauseNs[256-i:], circularPauseNs[:i])
	return linearPauseNs
}

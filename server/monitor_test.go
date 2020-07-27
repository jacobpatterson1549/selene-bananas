package server

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestGroutineExpectations(t *testing.T) {
	var servers [2]Server
	var numExpectations [2]int
	for i, httpAddr := range [2]string{"", ":8001"} {
		servers[i] = Server{
			httpServer: &http.Server{
				Addr: httpAddr,
			},
		}
	}
	for i, server := range servers {
		var w strings.Builder
		server.writeGoroutineExpectations(&w)
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

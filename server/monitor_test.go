package server

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestGroutineExpectations(t *testing.T) {
	var servers [2]Server
	for i, httpAddr := range []string{"", ":8001"} {
		servers[i] = Server{
			httpServer: &http.Server{
				Addr: httpAddr,
			},
		}
	}
	expectations := [2][]string{
		servers[0].goroutineExpectations(),
		servers[1].goroutineExpectations(),
	}
	for i, ei := range expectations {
		bulletCount := 0
		for _, e := range ei {
			if strings.HasPrefix(e, "* ") {
				bulletCount++
			}
		}
		want := strconv.Itoa(bulletCount)
		if len(ei) < 1 || !strings.Contains(ei[0], want) {
			t.Errorf("server %v: wanted %v goroutine expectations", i, want)
		}
	}
	if len(expectations[0]) == len(expectations[1]) {
		t.Error("wanted different goroutine expectations for http-only and http/https server")
	}
}

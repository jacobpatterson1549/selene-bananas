package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/db"
	"github.com/jacobpatterson1549/selene-bananas/game/word"
)

// TestNewWordValidator loads the embedded words, which should be the dump of the aspell en_US dictionary, Debian: aspell-en2018.04.16-0-1,Alpine: aspell-en=2020.12.07-r0
func TestNewWordValidator(t *testing.T) {
	r := strings.NewReader(embeddedWords)
	c := word.NewValidator(r)
	want := 77808
	got := len(*c)
	if want != got {
		note := "NOTE: this might be flaky, but it ensures that a large number of words can be loaded."
		t.Errorf("wanted %v words, got %v\n%v", want, got, note)
	}
}

// TestServerGet ensures the server handles get responses for files and templates.
// This integration test is SLOW because it starts a test server.
func TestServerGetFiles(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	log := log.New(&buf, "", 0)
	db := db.Database{}
	embedVersion = "1" // the version is not generated until the tests pass
	e, err := unembedData()
	if err != nil {
		t.Fatalf("unwanted error: %v", err)
	}
	f := flags{
		httpsPort: 8000, // not actually used, overridden by httptest
	}
	s, err := f.createServer(ctx, log, &db, *e)
	if err != nil {
		t.Fatalf("unwanted error: %v", err)
	}
	getHandler := s.HTTPSServer.Handler
	ts := httptest.NewTLSServer(getHandler)
	ts.Config.ErrorLog = log
	defer ts.Close()
	c := ts.Client()
	getFilePaths := []string{
		"/wasm_exec.js", "/main.wasm", "/robots.txt", "/favicon.png", "/LICENSE", // static files
		"/", "/manifest.json", "/serviceWorker.js", "/favicon.svg", "/network_check.html", // templates
	}
	for i, p := range getFilePaths {
		url := ts.URL + p
		res, err := c.Get(url)
		switch {
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case res.StatusCode != 200:
			body, _ := io.ReadAll(res.Body)
			res.Body.Close()
			t.Errorf("Test %v: wanted 200 code, got %v (%v): %s", i, res.StatusCode, res.Status, body)
		case buf.Len() != 0:
			t.Errorf("Test %v: unwanted log message, likely error: %v", i, buf.String())
			buf.Reset()
		}
	}
}

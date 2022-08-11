// Package main_test contains integration tests using embedded data.  Some of the tests are slow.
package main_test

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"net/http/httptest"
	"testing"

	main "github.com/jacobpatterson1549/selene-bananas/cmd/server"
	"github.com/jacobpatterson1549/selene-bananas/db"
	"github.com/jacobpatterson1549/selene-bananas/game/word"
	"github.com/jacobpatterson1549/selene-bananas/server/log/logtest"
)

func embeddedData(t *testing.T) main.EmbeddedData {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping long running test that uses embedded file system")
	}
	e, err := main.UnembedFS(main.EmbeddedFS)
	if err != nil {
		t.Fatalf("unembedding data: %v", err)
	}
	return *e
}

// TestNewWordValidator loads the embedded words, which should be the dump of the aspell en_US dictionary, Debian: aspell-en2018.04.16-0-1,Alpine: aspell-en=2020.12.07-r0
func TestNewWordValidator(t *testing.T) {
	t.Skip("test is flaky")
	e := embeddedData(t)
	r := bytes.NewReader(e.Words)
	validator, err := word.NewValidator(r)
	if err != nil {
		t.Fatalf("unwanted new word validate error: %v", err)
	}
	want := 77976
	got := len(*validator)
	if want != got {
		note := "NOTE: this might be flaky, but it ensures that a large number of words can be loaded."
		t.Errorf("wanted %v words, got %v\n%v", want, got, note)
	}
}

// TestServerGet ensures the server handles get responses for files and templates.
// This integration test is SLOW because it starts a test server.
func TestServerGetFiles(t *testing.T) {
	ctx := context.Background()
	log := logtest.DiscardLogger
	db := new(db.Database)
	e := embeddedData(t)
	f := main.Flags{
		HTTPSPort: 8000, // not actually used, overridden by httptest
	}
	s, err := f.CreateServer(ctx, log, db, e)
	if err != nil {
		t.Fatalf("unwanted create server error: %v", err)
	}
	getHandler := s.HTTPSServer.Handler
	ts := httptest.NewTLSServer(getHandler)
	defer ts.Close()
	c := ts.Client()
	getFilePaths := []string{
		"/wasm_exec.js", "/selene-bananas.wasm", "/robots.txt", "/favicon.png", "/favicon.ico", "/LICENSE", // static files
		"/", "/manifest.json", "/serviceWorker.js", "/favicon.svg", "/network_check.html", // templates
	}
	for i, p := range getFilePaths {
		url := ts.URL + p
		res, err := c.Get(url)
		switch {
		case err != nil:
			t.Errorf("Test %v: unwanted server get error: %v", i, err)
		case res.StatusCode != 200:
			body, err := io.ReadAll(res.Body)
			res.Body.Close()
			if err != nil {
				t.Errorf("reading error response body: %v", err)
			}
			t.Errorf("Test %v: wanted 200 code, got %v (%v): %s", i, res.StatusCode, res.Status, body)
		}
	}
}

func init() {
	sql.Register("TestDatabaseFilesExistDriver", main.TestNoopDriver)
}

func TestDatabaseFilesExist(t *testing.T) {
	var f main.Flags
	ctx := context.Background()
	e := embeddedData(t)
	if _, err := f.CreateDatabase(ctx, "TestDatabaseFilesExistDriver", e); err != nil {
		t.Errorf("unwanted create database error: %v", err)
	}
}

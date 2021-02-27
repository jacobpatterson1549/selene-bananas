package main

import (
	"context"
	"io"
	"log"
	"strings"
	"testing"
	"testing/fstest"

	databaseController "github.com/jacobpatterson1549/selene-bananas/db/sql"
)

// TestNewServer only checks the happy path, making sure defaults defined in config.go are valid.
func TestNewServer(t *testing.T) {
	m := mainFlags{
		httpsPort: 443,
	}
	ctx := context.Background()
	log := log.New(io.Discard, "", 0)
	db := databaseController.Database{}
	e := embeddedData{
		Version:     "1",
		WordsReader: strings.NewReader("apple\nbanana\ncarrot"),
		StaticFS:    fstest.MapFS{},
		TemplateFS: fstest.MapFS{
			"file": &fstest.MapFile{},
		},
	}
	s, err := m.createServer(ctx, log, db, e)
	switch {
	case err != nil:
		t.Errorf("unwanted error: %v", err)
	case s == nil:
		t.Error("nil server created")
	}
}

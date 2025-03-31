package main

import (
	"context"
	"database/sql"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/db"
	"github.com/jacobpatterson1549/selene-bananas/db/user"
	"github.com/jacobpatterson1549/selene-bananas/server/log/logtest"
)

// init registers a mock driver that does nothing when executing transactions
func init() {
	sql.Register("TestCreateDatabaseDriver", TestNoopDriver)
}

// TestCreateUserBackend checks for database validity
func TestCreateUserBackend(t *testing.T) {
	t.Run("no database url", func(t *testing.T) {
		var f Flags
		var e EmbeddedData
		ctx := context.Background()
		ub, err := f.CreateUserBackend(ctx, e)
		if err != nil {
			t.Errorf("unwanted error creating backend without url: %v", err)
		}
		if _, ok := ub.(user.NoDatabaseBackend); !ok {
			t.Errorf("wanted user.NoDatabaseBackend, got %T", ub)
		}
	})
	databaseURLs := []string{
		"postgres", // need more than this
		"ftp://",
	}
	for i, u := range databaseURLs {
		f := Flags{
			DatabaseURL: u,
		}
		var e EmbeddedData
		ctx := context.Background()
		if _, err := f.CreateUserBackend(ctx, e); err == nil {
			t.Errorf("Test %v: wanted err for %q database url", i, u)
		}
	}
}

// TestCreateSQLDatabase only checks the happy path, making sure defaults defined in config.go are valid.
func TestCreateSQLDatabase(t *testing.T) {
	var f Flags
	cfg := db.Config{
		QueryPeriod: 15 * time.Millisecond, // should take nearly no time
	}
	e := EmbeddedData{
		SQLFS: fstest.MapFS{
			"user_create.sql":                  &fstest.MapFile{Data: []byte("2")},
			"user_delete.sql":                  &fstest.MapFile{Data: []byte("6")},
			"user_read.sql":                    &fstest.MapFile{Data: []byte("3")},
			"users.sql":                        &fstest.MapFile{Data: []byte("1")},
			"user_update_password.sql":         &fstest.MapFile{Data: []byte("4")},
			"user_update_points_increment.sql": &fstest.MapFile{Data: []byte("5")},
		},
	}
	ctx := context.Background()
	got, err := f.CreateSQLDatabase(ctx, cfg, "TestCreateDatabaseDriver", e)
	switch {
	case err != nil:
		t.Errorf("unwanted error: %v", err)
	case got == nil:
		t.Error("nil database created")
	}
}

// TestCreateServer mainly checks the happy path, making sure defaults defined in config.go are valid.
func TestCreateServer(t *testing.T) {
	f := Flags{
		HTTPSPort: 443,
	}
	ctx := context.Background()
	log := logtest.DiscardLogger
	var ub mockUserBackend
	wantVersion := "9d2ffad8e5e5383569d37ec381147f2d"
	staticFS := new(fstest.MapFS)
	dummyFile := new(fstest.MapFile)
	e := EmbeddedData{
		Version:    []byte(wantVersion + "\n"),
		Words:      []byte("apple\nbanana\ncarrot"),
		StaticFS:   staticFS,
		TemplateFS: fstest.MapFS{"file": dummyFile},
	}
	s, err := f.CreateServer(ctx, log, ub, e)
	switch {
	case err != nil:
		t.Errorf("unwanted error: %v", err)
	case s == nil:
		t.Error("nil server created")
	case wantVersion != s.Config.Version:
		t.Errorf("config versions not equal:\nwanted: '%v'\ngot:    '%v'", wantVersion, s.Config.Version)
	}
}

func TestGameConfig(t *testing.T) {
	tests := []bool{
		true,
		false,
	}
	for _, debugGame := range tests {
		timeFunc := func() int64 {
			return 42
		}
		var f Flags
		f.DebugGame = debugGame
		cfg := f.gameConfig(timeFunc)
		if want, got := debugGame, cfg.Debug; want != got {
			t.Errorf("debug game flag not preserved when initially %v", want)
		}
	}
}

func TestDatabaseConfig(t *testing.T) {
	f := Flags{
		DBTimeoutSec: 8,
	}
	want := db.Config{
		QueryPeriod: 8 * time.Second,
	}
	got := f.databaseConfig()
	if want != got {
		t.Errorf("not equal:\nwanted: '%v'\ngot:    '%v'", want, got)
	}
}

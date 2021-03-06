package main

import (
	"context"
	"database/sql"
	"testing"
	"testing/fstest"

	"github.com/jacobpatterson1549/selene-bananas/db"
	"github.com/jacobpatterson1549/selene-bananas/server/log/logtest"
)

// init registers a mock driver that does nothing when executing transactions
func init() {
	sql.Register("TestCreateDatabaseDriver", TestNoopDriver)
}

// TestCreateDatabase only checks the happy path, making sure defaults defined in config.go are valid.
func TestCreateDatabase(t *testing.T) {
	var f Flags
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
	got, err := f.CreateDatabase(ctx, "TestCreateDatabaseDriver", e)
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
	db := db.Database{}
	wantVersion := "9d2ffad8e5e5383569d37ec381147f2d"
	e := EmbeddedData{
		Version:    []byte(wantVersion + "\n"),
		Words:      []byte("apple\nbanana\ncarrot"),
		StaticFS:   fstest.MapFS{},
		TemplateFS: fstest.MapFS{"file": &fstest.MapFile{}},
	}
	s, err := f.CreateServer(ctx, log, &db, e)
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
	t.Run("timeFunc seeds shufflers", func(t *testing.T) {
		timeFuncCalled := false
		timeFunc := func() int64 {
			timeFuncCalled = true
			return 42
		}
		var f Flags
		f.gameConfig(timeFunc)
		if !timeFuncCalled {
			t.Error("wanted timeFunc to be called to seed game shuffle funcs")
		}
	})
}

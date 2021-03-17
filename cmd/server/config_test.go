package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"log"
	"testing"
	"testing/fstest"

	"github.com/jacobpatterson1549/selene-bananas/db"
)

// init registers a mock driver that does nothing when executing transactions
func init() {
	sql.Register("TestCreateDatabaseDB", &mockDriver{
		OpenFunc: func(name string) (driver.Conn, error) {
			return mockConn{
				PrepareFunc: func(query string) (driver.Stmt, error) {
					return mockStmt{
						CloseFunc: func() error {
							return nil
						},
						NumInputFunc: func() int {
							return 0
						},
						ExecFunc: func(args []driver.Value) (driver.Result, error) {
							return nil, nil
						},
					}, nil
				},
				BeginFunc: func() (driver.Tx, error) {
					return mockTx{
						CommitFunc: func() error {
							return nil
						},
					}, nil
				},
			}, nil
		},
	})
}

// TestCreateDatabase only checks the happy path, making sure defaults defined in config.go are valid.
func TestCreateDatabase(t *testing.T) {
	var f flags
	e := embeddedData{
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
	got, err := f.createDatabase(ctx, "TestCreateDatabaseDB", e)
	switch {
	case err != nil:
		t.Errorf("unwanted error: %v", err)
	case got == nil:
		t.Error("nil database created")
	}
}

// TestNewServer only checks the happy path, making sure defaults defined in config.go are valid.
func TestNewServer(t *testing.T) {
	f := flags{
		httpsPort: 443,
	}
	ctx := context.Background()
	log := log.New(io.Discard, "", 0)
	db := db.Database{}
	e := embeddedData{
		Version:    "1",
		Words:      "apple\nbanana\ncarrot",
		StaticFS:   fstest.MapFS{},
		TemplateFS: fstest.MapFS{"file": &fstest.MapFile{}},
	}
	s, err := f.createServer(ctx, log, &db, e)
	switch {
	case err != nil:
		t.Errorf("unwanted error: %v", err)
	case s == nil:
		t.Error("nil server created")
	}
}

func TestGameConfig(t *testing.T) {
	t.Run("timeFunc seeds shufflers", func(t *testing.T) {
		timeFuncCalled := false
		timeFunc := func() int64 {
			timeFuncCalled = true
			return 42
		}
		var f flags
		f.gameConfig(timeFunc)
		if !timeFuncCalled {
			t.Error("wanted timeFunc to be called to seed game shuffle funcs")
		}
	})
}

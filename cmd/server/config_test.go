package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"log"
	"strings"
	"testing"
	"testing/fstest"

	databaseController "github.com/jacobpatterson1549/selene-bananas/db/sql"
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
	var m mainFlags
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
	got, err := m.createDatabase(ctx, "TestCreateDatabaseDB", e)
	switch {
	case err != nil:
		t.Errorf("unwanted error: %v", err)
	case got == nil:
		t.Error("nil database created")
	}
}

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

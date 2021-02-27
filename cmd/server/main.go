// Package main starts the server after configuring it from supplied or standard arguments
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jacobpatterson1549/selene-bananas/db"
	"github.com/jacobpatterson1549/selene-bananas/server"
	_ "github.com/lib/pq" // register "postgres" database driver from package init() function
)

// main configures and runs the server.
func main() {
	e, err := newEmbedParameters(embedVersion, embeddedWords, embeddedStaticFS, embeddedTemplateFS, embeddedSQLFS)
	if err != nil {
		log.Fatalf("reading embedded files: %v", err)
	}
	m := newMainFlags(os.Args, os.LookupEnv)
	ctx := context.Background()
	db, err := database(ctx, m, *e)
	logFlags := log.Ldate | log.Ltime | log.LUTC | log.Lshortfile | log.Lmsgprefix
	log := log.New(os.Stdout, "", logFlags)
	server, err := m.newServer(ctx, log, db, *e)
	if err != nil {
		log.Fatalf("creating server: %v", err)
	}
	runServer(ctx, server, log)
}

// runServer runs the server until it is interrupted or terminated.
func runServer(ctx context.Context, server *server.Server, log *log.Logger) {
	done := make(chan os.Signal, 2)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	errC := server.Run(ctx)
	select { // BLOCKING
	case err := <-errC:
		switch {
		case err == http.ErrServerClosed:
			log.Printf("server shutdown triggered")
		default:
			log.Printf("server stopped unexpectedly: %v", err)
		}
	case signal := <-done:
		log.Printf("handled signal: %v", signal)
	}
	if err := server.Stop(ctx); err != nil {
		log.Fatalf("stopping server: %v", err)
	}
	log.Println("server stopped successfully")
}

// database creates and sets up the database.
func database(ctx context.Context, m mainFlags, e embeddedData) (db.Database, error) {
	cfg := m.sqlDatabaseConfig()
	db, err := sql.Open("postgres", m.databaseURL)
	if err != nil {
		return nil, fmt.Errorf("opening database %w", err)
	}
	sqlDB, err := cfg.NewDatabase(db)
	if err != nil {
		return nil, fmt.Errorf("creating SQL database: %w", err)
	}
	setupSQL, err := e.sqlFiles()
	if err != nil {
		return nil, err
	}
	if err := sqlDB.Setup(ctx, setupSQL); err != nil {
		return nil, fmt.Errorf("setting up database: %w", err)
	}
	return sqlDB, nil
}

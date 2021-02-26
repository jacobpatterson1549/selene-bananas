// Package main starts the server after configuring it from supplied or standard arguments
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jacobpatterson1549/selene-bananas/db"
	"github.com/jacobpatterson1549/selene-bananas/server"
)

// main configures and runs the server.
func main() {
	e, err := newEmbedParameters(embedVersion, embeddedStaticFS, embeddedTemplateFS, embeddedSQLFS)
	if err != nil {
		log.Fatalf("reading embedded files: %v", err)
	}
	m := newMainFlags(os.Args, os.LookupEnv)
	wordsFile, err := os.Open(m.wordsFile)
	if err != nil {
		log.Fatalf("trying to open words file: %v", err)
	}
	ctx := context.Background()
	db, err := database(ctx, m, *e)
	logFlags := log.Ldate | log.Ltime | log.LUTC | log.Lshortfile | log.Lmsgprefix
	log := log.New(os.Stdout, "", logFlags)
	server, err := m.newServer(ctx, log, db, wordsFile, *e)
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
	sqlDatabaseConfig := m.sqlDatabaseConfig()
	db, err := sqlDatabaseConfig.NewDatabase()
	if err != nil {
		return nil, fmt.Errorf("creating SQL database: %w", err)
	}
	setupSQL, err := e.sqlFiles()
	if err != nil {
		return nil, err
	}
	db.Setup(ctx, setupSQL)
	return db, nil
}

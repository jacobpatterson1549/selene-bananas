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

	"github.com/jacobpatterson1549/selene-bananas/server"
	_ "github.com/lib/pq" // register "postgres" database driver from package init() function
)

// main configures and runs the server.
func main() {
	ctx := context.Background()
	logFlags := log.Ldate | log.Ltime | log.LUTC | log.Lshortfile | log.Lmsgprefix
	log := log.New(os.Stdout, "", logFlags)
	e, err := newEmbedParameters(embedVersion, embeddedWords, embeddedStaticFS, embeddedTemplateFS, embeddedSQLFS)
	if err != nil {
		log.Fatalf("reading embedded files: %v", err)
	}
	m := newMainFlags(os.Args, os.LookupEnv)
	db, err := m.createDatabase(ctx, "postgres", *e)
	if err != nil {
		log.Fatalf("setting up database: %v", err)
	}
	server, err := m.createServer(ctx, log, db, *e)
	if err != nil {
		log.Fatalf("creating server: %v", err)
	}
	err = runServer(ctx, server, log)
	if err != nil {
		log.Fatalf("running server: %v", err)
	}
	log.Println("server run stopped successfully")
}

// runServer runs the server until it is interrupted or terminated.
func runServer(ctx context.Context, server *server.Server, log *log.Logger) error {
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
		return fmt.Errorf("stopping server: %v", err)
	}
	return nil
}

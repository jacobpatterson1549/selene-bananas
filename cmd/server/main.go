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

	_ "github.com/lib/pq" // register "postgres" database driver from package init() function
)

// main configures and runs the server.
func main() {
	ctx := context.Background()
	logFlags := log.Ldate | log.Ltime | log.LUTC | log.Lshortfile | log.Lmsgprefix
	log := log.New(os.Stdout, "", logFlags)
	if err := runServer(ctx, log); err != nil {
		log.Fatal(err)
	}
}

// runServer runs the server
func runServer(ctx context.Context, log *log.Logger) error {
	e, err := UnembedFS(EmbeddedFS)
	if err != nil {
		return fmt.Errorf("reading embedded files: %v", err)
	}
	f := newFlags(os.Args, os.LookupEnv)
	ub, err := f.CreateUserBackend(ctx, *e)
	if err != nil {
		return fmt.Errorf("creating database: %v", err)
	}
	server, err := f.CreateServer(ctx, log, ub, *e)
	if err != nil {
		return fmt.Errorf("creating server: %v", err)
	}
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
	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("stopping server: %v", err)
	}
	log.Printf("server stopped successfully")
	return nil
}

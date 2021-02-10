// Package main starts the server after configuring it from supplied or standard arguments
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jacobpatterson1549/selene-bananas/server"
)

// main configures and runs the server.
func main() {
	m := newMainFlags(os.Args, os.LookupEnv)
	logFlags := log.Ldate | log.Ltime | log.LUTC | log.Llongfile | log.Lmsgprefix
	log := log.New(os.Stdout, "", logFlags)
	ctx := context.Background()
	server, err := newServer(ctx, m, log)
	if err != nil {
		log.Fatalf("creating server: %v", err)
	}
	runServer(ctx, *server, log)
}

// runServer runs the server until it is interrupted or terminated.
func runServer(ctx context.Context, server server.Server, log *log.Logger) {
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
		log.Printf("handled %v", signal)
	}
	if err := server.Stop(ctx); err != nil {
		log.Printf("stopping server: %v", err)
	}
}

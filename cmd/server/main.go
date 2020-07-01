// Package main starts the server after configuring it from supplied or standard arguments
package main

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	m := newMainFlags(os.Args, os.LookupEnv)

	var buf bytes.Buffer
	log := log.New(&buf, m.applicationName+" ", log.LstdFlags)
	log.SetOutput(os.Stdout)

	ctx := context.Background()
	cfg, err := serverConfig(ctx, m, log)
	if err != nil {
		log.Fatalf("configuring server: %v", err)
	}
	server, err := cfg.NewServer()
	if err != nil {
		log.Fatalf("creating server: %v", err)
	}

	done := make(chan os.Signal, 2)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		err := server.Run(ctx)
		if err != http.ErrServerClosed {
			log.Printf("server stopped unexpectedly: %v", err)
		}
		done <- syscall.SIGTERM
	}()
	<-done
	if err := server.Stop(ctx); err != nil {
		log.Fatalf("stopping server: %v", err)
	}
}

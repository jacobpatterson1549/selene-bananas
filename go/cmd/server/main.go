// Package main starts the server after configuring it from supplied or standard arguments
package main

import (
	"bytes"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
)

func main() {
	ctx := context.Background()

	m := newMainFlags(os.Args, os.LookupEnv)

	var buf bytes.Buffer
	log := log.New(&buf, m.applicationName+" ", log.LstdFlags)
	log.SetOutput(os.Stdout)

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
	if err := server.Run(ctx); err != nil {
		log.Fatalf("running server: %v", err)
	}

	<-done
	if err := server.Stop(ctx); err != nil {
		log.Fatalf("stopping server: %v", err)
	}
}
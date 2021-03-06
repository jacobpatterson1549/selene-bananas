// Package main starts the server after configuring it from supplied or standard arguments
package main

import (
	"context"
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
	e, err := unembedData()
	if err != nil {
		log.Fatalf("reading embedded files: %v", err)
	}
	f := newFlags(os.Args, os.LookupEnv)
	db, err := f.createDatabase(ctx, "postgres", *e)
	if err != nil {
		log.Fatalf("creating database: %v", err)
	}
	server, err := f.createServer(ctx, log, db, *e)
	if err != nil {
		log.Fatalf("creating server: %v", err)
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
	if err := server.Stop(ctx); err != nil {
		log.Fatalf("stopping server: %v", err)
	}
	log.Println("server stopped successfully")
}

// unembedData returns the unembedded data that was embedded in the server.
func unembedData() (*embeddedData, error) {
	e := embeddedData{
		Version:    embedVersion,
		Words:      embeddedWords,
		TLSCertPEM: embeddedTLSCertPEM,
		TLSKeyPEM:  embeddedTLSKeyPEM,
		StaticFS:   embeddedStaticFS,
		TemplateFS: embeddedTemplateFS,
		SQLFS:      embeddedSQLFS,
	}
	return e.unEmbed()
}

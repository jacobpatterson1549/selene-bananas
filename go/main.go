package main

import (
	"bytes"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	m := newMainFlags(os.Args)

	var buf bytes.Buffer
	log := log.New(&buf, m.applicationName+" ", log.LstdFlags)
	log.SetOutput(os.Stdout)

	cfg, err := serverConfig(m, log)
	if err != nil {
		log.Fatalf("configuring server: %v", err)
	}

	server, err := cfg.NewServer()
	if err != nil {
		log.Fatalf("creating server: %v", err)
	}

	err = server.Run()
	if err != nil {
		log.Fatalf("running server: %v", err)
	}
}

// Package server runs the http server with allows users to open websockets to play the game
package server

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

// Config contains fields which describe the server
type Config struct {
	port     string
	maxGames int
}

// Run starts the server
func Run() {
	err := handleStaticFolder()
	if err != nil {
		return err
	}
	http.HandleFunc("/", handleRoot(cfg))
	addr := fmt.Sprintf(":%s", cfg.port)
	cfg.log.Println("starting server - locally running at http://127.0.0.1" + addr)
	err = http.ListenAndServe(addr, nil) // BLOCKS
	if err != http.ErrServerClosed {
		return fmt.Errorf("server stopped unexpectedly: %w", err)
	}
	return nil
}

func handleStaticFolder(cfg Config) error {
	fileInfo, err := ioutil.ReadDir("static")
	if err != nil {
		return fmt.Errorf("reading static dir: %w", err)
	}
	for _, file := range fileInfo {
		path := "/" + file.Name()
		http.HandleFunc(path, handleStatic)
	}
	return nil
}

func handleStatic(w http.ResponseWriter, r *http.Request) {
	path := "static" + r.URL.Path
	http.ServeFile(w, r, path)
}

func handleRoot(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

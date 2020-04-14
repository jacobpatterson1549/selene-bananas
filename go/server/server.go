// Package server runs the http server with allows users to open websockets to play the game
package server

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/jacobpatterson1549/selene-bananas/go/server/db"

	"github.com/gorilla/websocket"
)

// Config contains fields which describe the server
type Config struct {
	ApplicationName string
	port            string
	userDao         db.UserDao
	log             *log.Logger
	upgrader        *websocket.Upgrader
}

// NewConfig creates a new configuration object for the server
func NewConfig(applicationName, port string, userDao db.UserDao, log *log.Logger) Config {
	return Config{
		ApplicationName: applicationName,
		port:            port,
		userDao:         userDao,
		log:             log,
		upgrader:        new(websocket.Upgrader),
	}
}

// Run starts the server
func Run(cfg Config) error {
	err := handleStaticFolder()
	if err != nil {
		return err
	}
	http.HandleFunc("/", cfg.handleMethod)

	addr := fmt.Sprintf(":%s", cfg.port)
	cfg.log.Println("starting server - locally running at http://127.0.0.1" + addr)
	err = http.ListenAndServe(addr, nil) // BLOCKS
	if err != http.ErrServerClosed {
		return fmt.Errorf("server stopped unexpectedly: %w", err)
	}
	return nil
}

func handleStaticFolder() error {
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

func (cfg Config) handleMethod(w http.ResponseWriter, r *http.Request) {
	var err error
	switch r.Method {
	case "GET":
		err = cfg.handleGet(w, r)
	case "POST":
		err = cfg.handlePost(w, r)
	default:
		http.Error(w, err.Error(), http.StatusMethodNotAllowed)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (cfg Config) handleGet(w http.ResponseWriter, r *http.Request) error {
	switch r.URL.Path {
	case "/user_login":
		err := cfg.handleWebSocket(w, r)
		if err != nil {
			return fmt.Errorf("websocket error: %w", err)
		}
	default:
		err := cfg.handleTemplate(w, r)
		if err != nil {
			return fmt.Errorf("rendering template: %w", err)
		}
	}
	return nil
}

func (cfg Config) handlePost(w http.ResponseWriter, r *http.Request) error {
	switch r.URL.Path {
	case "user_create":
		err := r.ParseForm()
		if err != nil {
			return fmt.Errorf("parsing form: %w", err)
		}
	default:
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}
	return nil
}

func (cfg Config) handleTemplate(w http.ResponseWriter, r *http.Request) error {
	var fileName string
	switch r.URL.Path {
	case "/":
		fileName = "html/game/content.html"
	case "/user_create":
		fileName = "html/user_create/content.html"
	case "/about":
		fileName = "html/about/content.html"
	default:
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return nil
	}
	t, err := template.ParseFiles("html/main.html", fileName)
	if err != nil {
		return err
	}
	// TODO: limit what is available on cfg
	return t.Execute(w, cfg)
}

func (cfg Config) handleWebSocket(w http.ResponseWriter, r *http.Request) error {
	conn, err := cfg.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return fmt.Errorf("upgrading to websocket connection: %w", err)
	}
	defer conn.Close()
	for {
		// messageType, messageBytes, err := conn.ReadMessage()
		var m Message
		err := conn.ReadJSON(&m)
		if err != nil {
			// TODO: handle close message and expected messages
			return fmt.Errorf("reading message: %w", err)
		}
		err = m.handle()
		if err != nil {
			return fmt.Errorf("handling action: %w", err)
		}
	}
}

// Package server runs the http server with allows users to open websockets to play the game
package server

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
	"github.com/jacobpatterson1549/selene-bananas/go/server/game"
)

type (
	// Config contains fields which describe the server
	Config struct {
		appName string
		port    string
		db      db.Database
		log     *log.Logger
	}

	// Server can be run to serve the site
	Server interface {
		// Run starts the server
		Run() error
	}

	server struct {
		data              templateData
		addr              string
		log               *log.Logger
		handler           http.Handler
		staticFileHandler http.Handler
		upgrader          *websocket.Upgrader
		lobby             game.Lobby
		userDao           db.UserDao
		tokenizer         Tokenizer
	}

	templateData struct {
		ApplicationName string
	}
)

// NewConfig creates a new configuration object for a Server
func NewConfig(appName, port string, db db.Database, log *log.Logger) Config {
	return Config{
		appName: appName,
		port:    port,
		db:      db,
		log:     log,
	}
}

// NewServer creates a Server from the Config
func (cfg Config) NewServer() (Server, error) {
	data := templateData{
		ApplicationName: cfg.appName,
	}
	addr := fmt.Sprintf(":%s", cfg.port)
	serveMux := new(http.ServeMux)
	staticFileHandler := http.FileServer(http.Dir("./static"))
	tokenizer, err := newTokenizer()
	if err != nil {
		cfg.log.Fatal(err)
	}
	lobby := game.NewLobby(cfg.log)
	userDao := db.NewUserDao(cfg.db)
	err = userDao.Setup()
	if err != nil {
		cfg.log.Fatal(err)
	}
	s := server{
		data:              data,
		addr:              addr,
		log:               cfg.log,
		handler:           serveMux,
		staticFileHandler: staticFileHandler,
		lobby:             lobby,
		userDao:           userDao,
		tokenizer:         tokenizer,
	}
	serveMux.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("js"))))
	serveMux.HandleFunc("/", s.httpMethodHandler)
	return s, nil
}

func (s server) Run() error {
	httpServer := &http.Server{
		Addr:    s.addr,
		Handler: s.handler,
	}
	s.log.Println("starting server - locally running at http://127.0.0.1" + httpServer.Addr)
	err := httpServer.ListenAndServe() // BLOCKS
	if err != http.ErrServerClosed {
		return fmt.Errorf("server stopped unexpectedly: %w", err)
	}
	return nil
}

func (s server) httpMethodHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	switch r.Method {
	case "GET":
		err = s.httpGetHandler(w, r)
	case "POST":
		err = s.httpPostHandler(w, r)
	case "PUT":
		err = httpUserChangeHandler(w, r, "/user_update_password", s.handleUserUpdatePassword)
	case "DELETE":
		err = httpUserChangeHandler(w, r, "/user_delete", s.handleUserDelete)
	default:
		http.Error(w, err.Error(), http.StatusMethodNotAllowed)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s server) httpGetHandler(w http.ResponseWriter, r *http.Request) error {
	switch r.URL.Path {
	case "/":
		err := s.handleTemplate(w, r)
		if err != nil {
			return fmt.Errorf("rendering template: %w", err)
		}
	case "/user_logout":
		handleUserLogout(w)
	default:
		s.staticFileHandler.ServeHTTP(w, r)
	}
	return nil
}

func (s server) httpPostHandler(w http.ResponseWriter, r *http.Request) error {
	var err error
	// TODO: shroud these user calls in httpUserChangeHandler()
	switch r.URL.Path {
	case "/user_create":
		err = s.handleUserCreate(w, r)
	case "/user_login":
		err = s.handleUserLogin(w, r)
	default:
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}
	return err
}

func httpUserChangeHandler(w http.ResponseWriter, r *http.Request, path string, fn userChangeFn) error {
	switch r.URL.Path {
	case path:
		err := fn(r)
		if err != nil {
			return err // TODO: the user should also be logged out here.  add tests for this. but set status to 4xx or 5xx
		}
		handleUserLogout(w)
	default:
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}
	return nil
}

func (s server) handleTemplate(w http.ResponseWriter, r *http.Request) error {
	t := template.New("main.html")
	templateFileGlobs := []string{
		"html/*.html",
		"html/**/*.html",
	}
	for _, g := range templateFileGlobs {
		_, err := t.ParseGlob(g)
		if err != nil {
			return err
		}
	}
	return t.Execute(w, s.data)
}

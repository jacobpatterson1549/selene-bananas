// Package server runs the http server with allows users to open websockets to play the game
package server

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/jacobpatterson1549/selene-bananas/go/server/db"

	"github.com/gorilla/websocket"
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
		appName           string
		addr              string
		log               *log.Logger
		handler           http.Handler
		staticFileHandler http.Handler
		userDao           db.UserDao
		upgrader          *websocket.Upgrader
	}
)

// NewConfig creates a new configuration object for a Server
func NewConfig(appName, port string, db db.Database, log *log.Logger) Config {
	upgrader := new(websocket.Upgrader)
	upgrader.Error = httpWebSocketHandlerError(log)
	return Config{
		appName: appName,
		port:    port,
		db:      db,
		log:     log,
	}
}

// NewServer creates a Server from the Config
func (cfg Config) NewServer() (Server, error) {
	addr := fmt.Sprintf(":%s", cfg.port)
	userDao := db.NewUserDao(cfg.db)
	err := userDao.Setup()
	if err != nil {
		log.Fatal(err)
	}
	serveMux := new(http.ServeMux)
	staticFileHandler := http.FileServer(http.Dir("./static"))
	s := server{
		appName:           cfg.appName,
		addr:              addr,
		log:               cfg.log,
		handler:           serveMux,
		staticFileHandler: staticFileHandler,
		userDao:           userDao,
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
		err = httpUserChangeHandler(w, r, "/user_create", s.handleUserCreate)
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
	case "/user_login":
		err := s.httpWebSocketHandler(w, r)
		if err != nil {
			return fmt.Errorf("websocket error: %w", err)
		}
	default:
		err := s.handleTemplate(w, r)
		if err != nil {
			return fmt.Errorf("rendering template: %w", err)
		}
	}
	return nil
}

func httpUserChangeHandler(w http.ResponseWriter, r *http.Request, path string, fn userChangeFn) error {
	switch r.URL.Path {
	case path:
		err := fn(r)
		if err != nil {
			return err
		}
		handleUserLogout(w)
	default:
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}
	return nil
}

func (s server) handleTemplate(w http.ResponseWriter, r *http.Request) error {
	filenames := make([]string, 2)
	filenames[0] = "html/main.html"
	switch r.URL.Path {
	case "/":
		filenames[1] = "html/game/content.html"
		filenames = append(filenames,
			"html/error_message.html",
			"html/game/user_update_password.html",
			"html/game/user_login.html",
			"html/game/user_delete.html",
			"html/user_input/username.html",
			"html/user_input/password.html",
			"html/user_input/password_confirm.html")
	case "/user_create":
		filenames[1] = "html/user_create/content.html"
		filenames = append(filenames,
			"html/error_message.html",
			"html/user_input/username.html",
			"html/user_input/password_confirm.html")
	case "/about":
		filenames[1] = "html/about/content.html"
	default:
		s.staticFileHandler.ServeHTTP(w, r)
		return nil
	}
	t, err := template.ParseFiles(filenames...)
	if err != nil {
		return err
	}
	data := struct {
		ApplicationName string
	}{
		ApplicationName: s.appName,
	}
	return t.Execute(w, data)
}

func (s server) httpWebSocketHandler(w http.ResponseWriter, r *http.Request) error {
	conn, err := s.upgrader.Upgrade(w, r, nil)
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

func httpWebSocketHandlerError(log *log.Logger) func(w http.ResponseWriter, r *http.Request, status int, reason error) {
	return func(w http.ResponseWriter, r *http.Request, status int, reason error) {
		log.Println(reason)
	}
}

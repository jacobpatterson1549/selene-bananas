// Package server runs the http server with allows users to open websockets to play the game
package server

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/jacobpatterson1549/selene-bananas/go/db"
	"github.com/jacobpatterson1549/selene-bananas/go/game/lobby"
)

type (
	// Server runs the site
	Server struct {
		data              interface{}
		addr              string
		log               *log.Logger
		handler           http.Handler
		staticFileHandler http.Handler
		tokenizer         Tokenizer
		userDao           db.UserDao
		lobby             *lobby.Lobby
	}

	// Config contains fields which describe the server
	Config struct {
		// AppName is the display name of the application
		AppName string
		// Port is the port number to run the server on
		Port string
		// Log is used to log errors and other information
		Log *log.Logger
		// Tokenizer is used to generate and parse session tokens
		Tokenizer Tokenizer
		// UserDao is used to track different users
		UserDao db.UserDao
		// LobbyCfg is used to create a game lobby
		LobbyCfg lobby.Config
	}
)

// NewServer creates a Server from the Config
func (cfg Config) NewServer() (*Server, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	data := struct {
		ApplicationName string
		Description     string
	}{
		ApplicationName: cfg.AppName,
		Description:     "a tile-based word-forming game",
	}
	addr := fmt.Sprintf(":%s", cfg.Port)
	serveMux := new(http.ServeMux)
	lobby, err := cfg.LobbyCfg.NewLobby()
	if err != nil {
		return nil, err
	}
	s := Server{
		data:      data,
		addr:      addr,
		log:       cfg.Log,
		handler:   serveMux,
		tokenizer: cfg.Tokenizer,
		userDao:   cfg.UserDao,
		lobby:     lobby,
	}
	serveMux.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("js"))))
	serveMux.HandleFunc("/", s.httpMethodHandler)
	return &s, nil
}

func (cfg Config) validate() error {
	switch {
	case len(cfg.AppName) == 0:
		return fmt.Errorf("application name required")
	case len(cfg.Port) == 0:
		return fmt.Errorf("port number required")
	case cfg.Log == nil:
		return fmt.Errorf("log required")
	case cfg.Tokenizer == nil:
		return fmt.Errorf("tokenizer required")
	case cfg.UserDao == nil:
		return fmt.Errorf("user dao required")
	}
	return nil
}

// Run starts the server
// The server runs until it receives a shutdown signal.  This function blocks.
func (s Server) Run() error {
	httpServer := &http.Server{
		Addr:    s.addr,
		Handler: s.handler,
	}
	done := make(chan struct{})
	s.lobby.Run(done)
	s.log.Println("starting server - locally running at http://127.0.0.1" + httpServer.Addr)
	err := httpServer.ListenAndServe() // BLOCKS
	done <- struct{}{}
	if err != http.ErrServerClosed {
		return fmt.Errorf("server stopped unexpectedly: %w", err)
	}
	return nil
}

func (s Server) httpMethodHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	switch r.Method {
	case "GET":
		err = s.httpGetHandler(w, r)
	case "POST":
		err = s.httpPostHandler(w, r)
	case "PUT":
	default:
		httpError(w, http.StatusMethodNotAllowed)
	}
	if err != nil {
		s.log.Printf("server error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s Server) httpGetHandler(w http.ResponseWriter, r *http.Request) error {
	switch r.URL.Path {
	case "/":
		err := s.handleTemplate(w, r)
		if err != nil {
			return fmt.Errorf("rendering template: %w", err)
		}
	case "/favicon.ico", "/robots.txt":
		http.ServeFile(w, r, "static"+r.URL.Path)
	case "/user_join_lobby":
		err := r.ParseForm()
		if err != nil {
			return fmt.Errorf("parsing form: %w", err)
		}
		tokenString := r.FormValue("access_token")
		tokenUsername, err := s.tokenizer.ReadUsername(tokenString)
		if err != nil {
			s.log.Printf("reading username from token: %v", err)
			httpError(w, http.StatusUnauthorized)
			return nil
		}
		return s.handleUserJoinLobby(w, r, tokenUsername)
	case "/ping":
		_, err := s.readTokenUsername(r)
		if err != nil {
			s.log.Print(err)
			httpError(w, http.StatusForbidden)
			return nil
		}
	default:
		httpError(w, http.StatusNotFound)
	}
	return nil
}

func (s Server) httpPostHandler(w http.ResponseWriter, r *http.Request) error {
	var tokenUsername string
	var err error
	switch r.URL.Path {
	case "/user_create", "/user_login":
	default:
		tokenUsername, err = s.readTokenUsername(r)
		if err != nil {
			s.log.Print(err)
			httpError(w, http.StatusForbidden)
			return nil
		}
	}
	switch r.URL.Path {
	case "/user_create":
		err = s.handleUserCreate(w, r)
	case "/user_login":
		err = s.handleUserLogin(w, r)
	case "/user_update_password":
		err = s.handleUserUpdatePassword(w, r, tokenUsername)
	case "/user_delete":
		err = s.handleUserDelete(w, r, tokenUsername)
	default:
		httpError(w, http.StatusNotFound)
	}
	return err
}

func (s Server) handleTemplate(w http.ResponseWriter, r *http.Request) error {
	t := template.New("main.html")
	templateFileGlobs := []string{
		"html/*.html",
		"html/**/*.html",
		"svg/*.svg",
		"static/main.css",
		"js/*.js",
	}
	for _, g := range templateFileGlobs {
		_, err := t.ParseGlob(g)
		if err != nil {
			return err
		}
	}
	return t.Execute(w, s.data)
}

func httpError(w http.ResponseWriter, statusCode int) {
	http.Error(w, http.StatusText(statusCode), statusCode)
}

func (s Server) addAuthorization(w http.ResponseWriter, u db.User) error {
	token, err := s.tokenizer.Create(u)
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(token))
	if err != nil {
		return fmt.Errorf("writing authorization token: %w", err)
	}
	return nil
}

func (s Server) readTokenUsername(r *http.Request) (string, error) {
	authorization := r.Header.Get("Authorization")
	if len(authorization) < 7 || authorization[:7] != "Bearer " {
		return "", fmt.Errorf("invalid authorization header: %v", authorization)
	}
	tokenString := authorization[7:]
	tokenUsername, err := s.tokenizer.ReadUsername(tokenString)
	if err != nil {
		return "", err
	}
	err = r.ParseForm()
	if err != nil {
		return "", fmt.Errorf("parsing form: %w", err)
	}
	formUsername := r.FormValue("username")
	if string(tokenUsername) != formUsername {
		return "", fmt.Errorf("username not same as token username")
	}
	return tokenUsername, nil
}

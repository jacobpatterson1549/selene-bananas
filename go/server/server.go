// Package server runs the http server with allows users to open websockets to play the game
package server

import (
	"compress/gzip"
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/go/db"
	"github.com/jacobpatterson1549/selene-bananas/go/game/lobby"
)

type (
	// Server runs the site
	Server struct {
		data       interface{}
		addr       string
		log        *log.Logger
		handler    http.Handler
		tokenizer  Tokenizer
		userDao    *db.UserDao
		lobby      *lobby.Lobby
		httpServer *http.Server
		stopDur    time.Duration
		cacheSec   int
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
		UserDao *db.UserDao
		// LobbyCfg is used to create a game lobby
		LobbyCfg lobby.Config
		// StopDur is the maximum duration the server should take to shutdown gracefully
		StopDur time.Duration
		// CachenSec is the number of seconds some files are cached
		CacheSec int
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
		stopDur:   cfg.StopDur,
		cacheSec:  cfg.CacheSec,
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
	case cfg.StopDur <= 0:
		return fmt.Errorf("shop timeout duration required")
	case cfg.CacheSec < 0:
		return fmt.Errorf("non-negative cache time required")
	}
	return nil
}

// Run runs the server until it receives a shutdown signal.
func (s *Server) Run(ctx context.Context) error {
	if s.httpServer != nil {
		return fmt.Errorf("server already run")
	}
	s.httpServer = &http.Server{
		Addr:    s.addr,
		Handler: s.handler,
	}
	lobbyCtx, lobbyCancelFunc := context.WithCancel(ctx)
	s.httpServer.RegisterOnShutdown(lobbyCancelFunc)
	go s.lobby.Run(lobbyCtx)
	s.log.Println("server started successfully, locally running at http://127.0.0.1" + s.addr)
	go s.httpServer.ListenAndServe()
	return nil
}

// Stop asks the server to shutdown and waits for the shutdown to complete.
// An error is returned if the server if the context times out.
func (s *Server) Stop(ctx context.Context) error {
	ctx, cancelFunc := context.WithTimeout(ctx, s.stopDur)
	defer cancelFunc()
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return err
	}
	s.log.Println("server stopped successfully")
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

func (s *Server) httpGetHandler(w http.ResponseWriter, r *http.Request) error {
	switch r.URL.Path {
	case "/":
		if err := s.handleTemplate(w, r); err != nil {
			return fmt.Errorf("rendering template : %w", err)
		}
		return nil
	case "/favicon.ico", "/robots.txt", "/run_wasm.js":
		return s.handleStaticFile(w, r)
	case "/wasm_exec.js":
		return s.handleRootFile(w, r)
	case "/main.wasm":
		return s.handleWasmFile(w, r)
	case "/lobby":
		return s.handleLobby(w, r)
	case "/ping":
		return s.handleHTTPPing(w, r)
	case "/monitor":
		return s.handleMonitor(w, r)
	default:
		httpError(w, http.StatusNotFound)
		return nil
	}
}

func (s Server) httpPostHandler(w http.ResponseWriter, r *http.Request) error {
	var tokenUsername string
	var err error
	switch r.URL.Path {
	case "/user_create", "/user_login":
		// [unauthenticated]
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
		"static/fa/*.svg",
		"static/main.css",
	}
	for _, g := range templateFileGlobs {
		_, err := t.ParseGlob(g)
		if err != nil {
			return err
		}
	}
	return t.Execute(w, s.data)
}

func (s *Server) cacheResponse(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", s.cacheSec))
}

func (s *Server) handleStaticFile(w http.ResponseWriter, r *http.Request) error {
	s.cacheResponse(w)
	http.ServeFile(w, r, "static"+r.URL.Path)
	return nil
}

func (s *Server) handleRootFile(w http.ResponseWriter, r *http.Request) error {
	s.cacheResponse(w)
	http.ServeFile(w, r, "."+r.URL.Path)
	return nil
}

func (s *Server) handleWasmFile(w http.ResponseWriter, r *http.Request) error {
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		gzw := gzip.NewWriter(w)
		defer gzw.Close()
		gzrw := gzipResponseWriter{
			Writer:         gzw,
			ResponseWriter: w,
		}
		w = http.ResponseWriter(gzrw)
	}
	s.cacheResponse(w)
	http.ServeFile(w, r, "."+r.URL.Path)
	return nil
}

func (s *Server) handleLobby(w http.ResponseWriter, r *http.Request) error {
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
}

func (s Server) handleHTTPPing(w http.ResponseWriter, r *http.Request) error {
	_, err := s.readTokenUsername(r)
	if err != nil {
		s.log.Print(err)
		httpError(w, http.StatusForbidden)
	}
	return nil
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

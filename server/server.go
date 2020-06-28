// Package server runs the http server with allows users to open websockets to play the game
package server

import (
	"compress/gzip"
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/db"
	"github.com/jacobpatterson1549/selene-bananas/game/lobby"
)

type (
	// Server runs the site
	Server struct {
		data       interface{}
		log        *log.Logger
		tokenizer  Tokenizer
		userDao    *db.UserDao
		lobby      *lobby.Lobby
		httpServer *http.Server
		stopDur    time.Duration
		cacheSec   int
		version    string
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
		// Version is used to bust caches of files from older server version
		Version string
	}

	// wrappedResponseWriter wraps response writing with another writer.
	wrappedResponseWriter struct {
		io.Writer
		http.ResponseWriter
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
		Version         string
	}{
		ApplicationName: cfg.AppName,
		Description:     "a tile-based word-forming game",
		Version:         cfg.Version,
	}
	addr := fmt.Sprintf(":%s", cfg.Port)
	serveMux := new(http.ServeMux)
	lobby, err := cfg.LobbyCfg.NewLobby()
	if err != nil {
		return nil, err
	}
	s := Server{
		data:      data,
		log:       cfg.Log,
		tokenizer: cfg.Tokenizer,
		userDao:   cfg.UserDao,
		lobby:     lobby,
		httpServer: &http.Server{
			Addr:    addr,
			Handler: serveMux,
		},
		stopDur:  cfg.StopDur,
		cacheSec: cfg.CacheSec,
		version:  cfg.Version,
	}
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
func (s Server) Run(ctx context.Context) {
	lobbyCtx, lobbyCancelFunc := context.WithCancel(ctx)
	s.httpServer.RegisterOnShutdown(lobbyCancelFunc)
	go s.lobby.Run(lobbyCtx)
	s.log.Println("server started successfully, locally running at http://127.0.0.1" + s.httpServer.Addr)
	go s.httpServer.ListenAndServe()
}

// Stop asks the server to shutdown and waits for the shutdown to complete.
// An error is returned if the server if the context times out.
func (s Server) Stop(ctx context.Context) error {
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

func (s Server) httpGetHandler(w http.ResponseWriter, r *http.Request) error {
	var err error
	switch r.URL.Path {
	case "/":
		s.handleFile(s.serveTemplate, false)(w, r)
	case "/favicon.ico", "/robots.txt":
		s.handleFile(s.serveFile("resources"+r.URL.Path), false)(w, r)
	case "/wasm_exec.js", "/main.wasm":
		s.handleFile(s.serveFile("."+r.URL.Path), true)(w, r)
	case "/lobby":
		err = s.handleLobby(w, r)
	case "/ping":
		err = s.handleHTTPPing(w, r)
	case "/monitor":
		err = s.handleMonitor(w, r)
	default:
		httpError(w, http.StatusNotFound)
	}
	return err
}

func (s Server) httpPostHandler(w http.ResponseWriter, r *http.Request) error {
	var err error
	var tokenUsername string
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

func (s Server) serveTemplate(w http.ResponseWriter, r *http.Request) {
	t := template.New("main.html")
	templateFileGlobs := []string{
		"resources/html/**/*.html",
		"resources/fa/*.svg",
		"resources/main.css",
		"resources/run_wasm.js",
	}
	for _, g := range templateFileGlobs {
		_, err := t.ParseGlob(g)
		if err != nil {
			s.log.Printf("globbing template files: %v", err)
			httpError(w, http.StatusInternalServerError)
			return
		}
	}
	if err := t.Execute(w, s.data); err != nil {
		s.log.Printf("rendering template: %v", err)
		httpError(w, http.StatusInternalServerError)
	}
}

// serveFile serves the file from the filesystem
func (Server) serveFile(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, name)
	}
}

// handleFile wraps the handling of the file, add cache-control header and gzip compression, if possible.
func (s Server) handleFile(fn http.HandlerFunc, checkVersion bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if checkVersion && r.URL.Query().Get("v") != s.version {
			url := r.URL
			q := url.Query()
			q.Set("v", s.version)
			url.RawQuery = q.Encode()
			w.Header().Set("Location", url.String())
			w.WriteHeader(http.StatusMovedPermanently)
			return
		}
		if s.cacheSec > 0 && !strings.Contains(r.Header.Get("Cache-Control"), "no-store") {
			w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", s.cacheSec))
		}
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			w2 := gzip.NewWriter(w)
			defer w2.Close()
			w = wrappedResponseWriter{
				Writer:         w2,
				ResponseWriter: w,
			}
		}
		fn(w, r)
	}
}

func (s Server) handleLobby(w http.ResponseWriter, r *http.Request) error {
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

func (wrw wrappedResponseWriter) Write(p []byte) (n int, err error) {
	return wrw.Writer.Write(p)
}

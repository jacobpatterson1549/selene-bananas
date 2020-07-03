// Package server runs the http server with allows users to open websockets to play the game
package server

import (
	"compress/gzip"
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/db"
	"github.com/jacobpatterson1549/selene-bananas/game/lobby"
)

type (
	// Server runs the site
	Server struct {
		data        interface{}
		log         *log.Logger
		tokenizer   Tokenizer
		userDao     *db.UserDao
		lobby       *lobby.Lobby
		httpsServer *http.Server
		httpServer  *http.Server
		stopDur     time.Duration
		cacheSec    int
		version     string
		challenge   Challenge
		tlsCertFile string
		tlsKeyFile  string
	}

	// Config contains fields which describe the server
	Config struct {
		// HTTPPort is the TCP port for server http requests.  All traffic is redirected to the https port.
		HTTPPort int
		// HTTPSPORT is the TCP port for server https requests.
		HTTPSPort int
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
		// Challenge is the ACME HTTP-01 Challenge used to get a certificate
		Challenge Challenge
		// The public HTTPS certificate file.
		TLSCertFile string
		// The private HTTPS key file.
		TLSKeyFile string
		// ColorConfig contains the colors to use on the site.
		ColorConfig ColorConfig
	}

	// ColorConfig represents the colors on the site.
	ColorConfig struct {
		// The color to paint text on the canvas.
		CanvasPrimary string
		// The color to paint text of tiles when they are bing dragged.
		CanvasDrag string
		// The color to paint tiles on the canvas.
		CanvasTile string
		// The color of log error messages.
		LogError string
		// The color of log warning messages.
		LogWarning string
		// The color of log chat messages between players.
		LogChat string
		// The color of the background of tabs.
		TabBackground string
		// The color of even-numbered columns in tables.
		TableStripe string
		// The color of a button.
		Button string
		// The color of a button when the mouse hovers over it.
		ButtonHover string
		// The color when a button is active (actually a tab).
		ButtonActive string
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
		Colors          ColorConfig
	}{
		ApplicationName: "selene-bananas",
		Description:     "a tile-based word-forming game",
		Version:         cfg.Version,
		Colors:          cfg.ColorConfig,
	}
	httpsAddr := fmt.Sprintf(":%d", cfg.HTTPSPort)
	httpAddr := fmt.Sprintf(":%d", cfg.HTTPPort)
	httpsServeMux := new(http.ServeMux)
	httpsServer := &http.Server{
		Addr:    httpsAddr,
		Handler: httpsServeMux,
	}
	httpServeMux := new(http.ServeMux)
	httpServer := &http.Server{
		Addr:    httpAddr,
		Handler: httpServeMux,
	}
	lobby, err := cfg.LobbyCfg.NewLobby()
	if err != nil {
		return nil, err
	}
	s := Server{
		data:        data,
		log:         cfg.Log,
		tokenizer:   cfg.Tokenizer,
		userDao:     cfg.UserDao,
		lobby:       lobby,
		httpsServer: httpsServer,
		httpServer:  httpServer,
		stopDur:     cfg.StopDur,
		cacheSec:    cfg.CacheSec,
		version:     cfg.Version,
		challenge:   cfg.Challenge,
		tlsCertFile: cfg.TLSCertFile,
		tlsKeyFile:  cfg.TLSKeyFile,
	}
	httpsServeMux.HandleFunc("/", s.handleHTTPS)
	httpServeMux.HandleFunc("/", s.handleHTTP)
	return &s, nil
}

func (cfg Config) validate() error {
	switch {
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

// Run the server until it receives a shutdown signal.
func (s Server) Run(ctx context.Context) error {
	lobbyCtx, lobbyCancelFunc := context.WithCancel(ctx)
	s.httpsServer.RegisterOnShutdown(lobbyCancelFunc)
	go s.lobby.Run(lobbyCtx)
	errC := make(chan error, 2)
	go func() {
		errC <- s.httpsServer.ListenAndServeTLS(s.tlsCertFile, s.tlsKeyFile)
	}()
	go func() {
		errC <- s.httpServer.ListenAndServe()
	}()
	s.log.Println("server started, running at https://127.0.0.1" + s.httpsServer.Addr)
	return <-errC
}

// Stop asks the server to shutdown and waits for the shutdown to complete.
// An error is returned if the server if the context times out.
func (s Server) Stop(ctx context.Context) error {
	ctx, cancelFunc := context.WithTimeout(ctx, s.stopDur)
	defer cancelFunc()
	httpsShutdownErr := s.httpsServer.Shutdown(ctx)
	httpShutdownErr := s.httpServer.Shutdown(ctx)
	switch {
	case httpsShutdownErr != nil:
		return httpsShutdownErr
	case httpShutdownErr != nil:
		return httpShutdownErr
	}
	s.log.Println("server stopped successfully")
	return nil
}

func (s Server) handleHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	if strings.Contains(host, ":") {
		var err error
		host, _, err = net.SplitHostPort(host)
		if err != nil {
			message := "could not redirect to https: " + err.Error()
			http.Error(w, message, http.StatusInternalServerError)
			return
		}
	}
	if s.httpsServer.Addr != ":443" {
		host = host + s.httpsServer.Addr
	}
	httpsURI := "https://" + host + r.URL.Path
	http.Redirect(w, r, httpsURI, http.StatusTemporaryRedirect)
}

func (s Server) handleHTTPS(w http.ResponseWriter, r *http.Request) {
	var err error
	switch r.Method {
	case "GET":
		err = s.handleHTTPSGet(w, r)
	case "POST":
		err = s.handleHTTPSPost(w, r)
	case "PUT":
	default:
		httpError(w, http.StatusMethodNotAllowed)
	}
	if err != nil {
		s.log.Printf("server error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s Server) handleHTTPSGet(w http.ResponseWriter, r *http.Request) error {
	var err error
	switch r.URL.Path {
	case "/", "/manifest.json":
		s.handleFile(s.serveTemplate(r.URL.Path), false)(w, r)
	case "/wasm_exec.js", "/main.wasm":
		s.handleFile(s.serveFile("."+r.URL.Path), true)(w, r)
	case "/robots.txt", "/init.js", "/service-worker.js", "/favicon.ico", "/favicon-192.png", "/favicon-512.png":
		s.handleFile(s.serveFile("resources"+r.URL.Path), false)(w, r)
	case "/lobby":
		err = s.handleLobby(w, r)
	case "/ping":
		err = s.handleHTTPPing(w, r)
	case "/monitor":
		err = s.handleMonitor(w, r)
	default:
		if s.challenge.isFor(r.URL.Path) {
			s.challenge.handle(w, r)
			return nil
		}
		httpError(w, http.StatusNotFound)
	}
	return err
}

func (s Server) handleHTTPSPost(w http.ResponseWriter, r *http.Request) error {
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

func (s Server) serveTemplate(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if name == "/" {
			name = "/main.html"
		}
		t := template.New(name[1:])
		switch name {
		case "/main.html":
			templateFileGlobs := []string{
				"resources/html/**/*.html",
				"resources/fa/*.svg",
				"resources/main.css",
				"resources/init.js",
			}
			for _, g := range templateFileGlobs {
				if _, err := t.ParseGlob(g); err != nil {
					s.log.Printf("globbing template files: %v", err)
					httpError(w, http.StatusInternalServerError)
					return
				}
			}
		case "/manifest.json":
			if _, err := t.ParseFiles("resources" + name); err != nil {
				s.log.Printf("parsing manifest template: %v", err)
				httpError(w, http.StatusInternalServerError)
				return
			}
		default:
			s.log.Printf("unknown template: %v", name)
			httpError(w, http.StatusInternalServerError)
			return
		}
		if err := t.Execute(w, s.data); err != nil {
			s.log.Printf("rendering template: %v", err)
			httpError(w, http.StatusInternalServerError)
		}
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

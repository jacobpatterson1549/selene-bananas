// Package server runs the http server with allows users to open websockets to play the game
package server

import (
	"compress/gzip"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/server/auth"
	"github.com/jacobpatterson1549/selene-bananas/server/certificate"
)

type (
	// Server runs the site
	Server struct {
		data          interface{}
		log           *log.Logger
		tokenizer     auth.Tokenizer
		userDao       UserDao
		lobby         Lobby
		httpsServer   *http.Server
		httpServer    *http.Server
		stopDur       time.Duration
		cacheSec      int
		version       string
		challenge     certificate.Challenge
		tlsCertFile   string
		tlsKeyFile    string
		noTLSRedirect bool
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
		Tokenizer auth.Tokenizer
		// UserDao is used to track different users
		UserDao UserDao
		// Lobby is used to let lusers play Games.
		Lobby Lobby
		// StopDur is the maximum duration the server should take to shutdown gracefully
		StopDur time.Duration
		// CachenSec is the number of seconds some files are cached
		CacheSec int
		// Version is used to bust caches of files from older server version
		Version string
		// Challenge is the ACME HTTP-01 Challenge used to get a certificate
		Challenge certificate.Challenge
		// The public HTTPS certificate file.
		TLSCertFile string
		// The private HTTPS key file.
		TLSKeyFile string
		// ColorConfig contains the colors to use on the site.
		ColorConfig ColorConfig
		// NoTLSRedirect disables redirection to https from http when true.
		NoTLSRedirect bool
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
		return nil, fmt.Errorf("creating server: validation: %w", err)
	}
	data := struct {
		Name        string
		ShortName   string
		Description string
		Version     string
		Colors      ColorConfig
	}{
		Name:        "selene-bananas",
		ShortName:   "bananas",
		Description: "a tile-based word-forming game",
		Version:     cfg.Version,
		Colors:      cfg.ColorConfig,
	}
	httpsAddr := fmt.Sprintf(":%d", cfg.HTTPSPort)
	if cfg.HTTPSPort <= 0 {
		return nil, fmt.Errorf("invalid https port: %v", cfg.HTTPSPort)
	}
	httpAddr := fmt.Sprintf(":%d", cfg.HTTPPort)
	if cfg.HTTPPort <= 0 {
		httpAddr = ""
	}
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
	s := Server{
		data:          data,
		log:           cfg.Log,
		tokenizer:     cfg.Tokenizer,
		userDao:       cfg.UserDao,
		lobby:         cfg.Lobby,
		httpsServer:   httpsServer,
		httpServer:    httpServer,
		stopDur:       cfg.StopDur,
		cacheSec:      cfg.CacheSec,
		version:       cfg.Version,
		challenge:     cfg.Challenge,
		tlsCertFile:   cfg.TLSCertFile,
		tlsKeyFile:    cfg.TLSKeyFile,
		noTLSRedirect: cfg.NoTLSRedirect,
	}
	httpsServeMux.HandleFunc("/", s.handleHTTPS)
	httpServeMux.HandleFunc("/", s.handleHTTP)
	return &s, nil
}

// validate ensures the configuration has no errors.
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

// Run the server asynchronously until it receives a shutdown signal.
// When the HTTP/HTTPS servers stop, errors are logged to the error channel.
func (s Server) Run(ctx context.Context) <-chan error {
	errC := make(chan error, 2)
	go s.runHTTPServer(ctx, errC)
	go s.runHTTPSServer(ctx, errC)
	return errC
}

// validHTTPAddr determines if the HTTP address is valid.
// If the HTTP address is valid, the HTTP server should be started to redirect to HTTPS and handle certificate creation.
func (s Server) validHTTPAddr() bool {
	return len(s.httpServer.Addr) > 0
}

// runHTTPSServer runs the http server, adding the return error to the channel when done.
// The server is not run if the HTTP address is not valid.
func (s Server) runHTTPServer(ctx context.Context, errC chan<- error) {
	if !s.validHTTPAddr() {
		return
	}
	errC <- s.httpServer.ListenAndServe()
}

// runHTTPSServer runs the https server in regards to the conviguration, adding the return error to the channel when done.
func (s Server) runHTTPSServer(ctx context.Context, errC chan<- error) {
	lobbyCtx, lobbyCancelFunc := context.WithCancel(ctx)
	go s.lobby.Run(lobbyCtx)
	s.httpsServer.RegisterOnShutdown(lobbyCancelFunc)
	s.log.Printf("starting https server at at https://127.0.0.1%v", s.httpsServer.Addr)
	switch {
	case s.validHTTPAddr():
		if _, err := tls.LoadX509KeyPair(s.tlsCertFile, s.tlsKeyFile); err != nil {
			s.log.Printf("Problem loading tls certificate: %v", err)
			return
		}
		errC <- s.httpsServer.ListenAndServeTLS(s.tlsCertFile, s.tlsKeyFile)
	default:
		if len(s.tlsCertFile) != 0 || len(s.tlsKeyFile) != 0 {
			s.log.Printf("Ignoring TLS_CERT_FILE/TLS_KEY_FILE variables since PORT was specified, using automated certificate management.")
		}
		errC <- s.httpsServer.ListenAndServe()
	}
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

// handleHTTPS handles http endpoints.
func (s Server) handleHTTP(w http.ResponseWriter, r *http.Request) {
	if s.challenge.IsFor(r.URL.Path) {
		if err := s.challenge.Handle(w, r.URL.Path); err != nil {
			s.log.Printf("serving acme challenge: %v", err)
			s.httpError(w, http.StatusInternalServerError)
		}
		return
	}
	host := r.Host
	if strings.Contains(host, ":") {
		var err error
		host, _, err = net.SplitHostPort(host)
		if err != nil {
			err = fmt.Errorf("could not redirect to https: %w", err)
			s.handleError(w, err)
			return
		}
	}
	if s.httpsServer.Addr != ":443" && !s.noTLSRedirect {
		host = host + s.httpsServer.Addr
	}
	httpsURI := "https://" + host + r.URL.Path
	http.Redirect(w, r, httpsURI, http.StatusTemporaryRedirect)
}

// handleHTTPS handles https endpoints.
func (s Server) handleHTTPS(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.TLS == nil && !s.noTLSRedirect:
		s.handleHTTP(w, r)
	case r.Method == "GET":
		s.handleHTTPSGet(w, r)
	case r.Method == "POST":
		s.handleHTTPSPost(w, r)
	default:
		s.httpError(w, http.StatusMethodNotAllowed)
	}
}

// handleHTTPSGet calls handlers for GET endpoints.
func (s Server) handleHTTPSGet(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/", "/manifest.json", "/serviceWorker.js":
		s.handleFile(w, r, s.serveTemplate(r.URL.Path), false)
	case "/wasm_exec.js", "/main.wasm":
		s.handleFile(w, r, s.serveFile("."+r.URL.Path), true)
	case "/robots.txt", "/favicon.svg", "/favicon.png":
		s.handleFile(w, r, s.serveFile("resources"+r.URL.Path), false)
	case "/lobby":
		s.handleUserLobby(w, r)
	case "/ping":
		s.handleHTTPPing(w, r)
	case "/monitor":
		s.handleMonitor(w, r)
	default:
		s.httpError(w, http.StatusNotFound)
	}
}

// handleHTTPSPost checks authentication and calls handlers for POST endpoints.
func (s Server) handleHTTPSPost(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user_create", "/user_login":
		// [unauthenticated]
	default:
		if err := s.checkTokenUsername(r); err != nil {
			s.log.Print(err)
			s.httpError(w, http.StatusForbidden)
			return
		}
	}
	switch r.URL.Path {
	case "/user_create":
		s.handleUserCreate(w, r)
	case "/user_login":
		s.handleUserLogin(w, r)
	case "/user_update_password":
		s.handleUserUpdatePassword(w, r)
	case "/user_delete":
		s.handleUserDelete(w, r)
	default:
		s.httpError(w, http.StatusNotFound)
	}
}

// sereveTemplate servers the file from the data-driven template.
func (s Server) serveTemplate(name string) http.HandlerFunc {
	var (
		t         *template.Template
		filenames []string
	)
	switch name {
	case "/":
		t = template.New("main.html")
		templateFileGlobs := []string{
			"resources/html/**/*.html",
			"resources/fa/*.svg",
			"resources/main.css",
			"resources/init.js",
		}
		for _, g := range templateFileGlobs {
			matches, err := filepath.Glob(g)
			if err != nil {
				return func(w http.ResponseWriter, r *http.Request) {
					s.handleError(w, err)
				}
			}
			filenames = append(filenames, matches...)
		}
	default:
		t = template.New(name[1:])
		filenames = append(filenames, "resources"+name)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := t.ParseFiles(filenames...); err != nil {
			err = fmt.Errorf("parsing manifest template: %v", err)
			s.handleError(w, err)
			return
		}
		if len(name) > 3 && name[len(name)-3:] == ".js" {
			w.Header().Set("Content-Type", "application/javascript")
		}
		if err := t.Execute(w, s.data); err != nil {
			err = fmt.Errorf("rendering template: %v", err)
			s.handleError(w, err)
			return
		}
	}
}

// serveFile serves the file from the filesystem.
func (Server) serveFile(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, name)
	}
}

// handleFile wraps the handling of the file, add cache-control header and gzip compression, if possible.
func (s Server) handleFile(w http.ResponseWriter, r *http.Request, fn http.HandlerFunc, checkVersion bool) {
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
		w2 := gzip.NewWriter(w)
		defer w2.Close()
		w = wrappedResponseWriter{
			Writer:         w2,
			ResponseWriter: w,
		}
		w.Header().Set("Content-Encoding", "gzip")
	}
	fn(w, r)
}

// handleHTTPPing ensures the ping is for a valid user.
func (s Server) handleHTTPPing(w http.ResponseWriter, r *http.Request) {
	if err := s.checkTokenUsername(r); err != nil {
		s.httpError(w, http.StatusUnauthorized)
	}
}

// httpError writes the error status code.
func (Server) httpError(w http.ResponseWriter, statusCode int) {
	http.Error(w, http.StatusText(statusCode), statusCode)
}

// handleError logs and writes the error as an internal server error (500).
func (s Server) handleError(w http.ResponseWriter, err error) {
	s.log.Printf("server error: %v", err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

// checkTokenUsername ensures the username in the authorization header matches that in the username form value.
func (s Server) checkTokenUsername(r *http.Request) error {
	authorization := r.Header.Get("Authorization")
	if len(authorization) < 7 || authorization[:7] != "Bearer " {
		return fmt.Errorf("invalid authorization header: %v", authorization)
	}
	tokenString := authorization[7:]
	tokenUsername, err := s.tokenizer.ReadUsername(tokenString)
	if err != nil {
		return err
	}
	formUsername := r.FormValue("username")
	if tokenUsername != formUsername {
		return fmt.Errorf("username not same as token username")
	}
	return nil
}

// Write delegates the write to the wrapped writer.
func (wrw wrappedResponseWriter) Write(p []byte) (n int, err error) {
	return wrw.Writer.Write(p)
}

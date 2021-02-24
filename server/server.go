// Package server runs the http server with allows users to open websockets to play the game
package server

import (
	"compress/gzip"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/server/certificate"
	"github.com/jacobpatterson1549/selene-bananas/server/game"
)

type (
	// Server runs the site
	Server struct {
		wg          sync.WaitGroup
		log         *log.Logger
		data        interface{}
		tokenizer   Tokenizer
		userDao     UserDao
		lobby       Lobby
		httpServer  *http.Server
		httpsServer *http.Server
		cacheMaxAge string
		template    *template.Template
		serveStatic http.Handler
		Config
	}

	// Config contains fields which describe the server
	Config struct {
		// HTTPPort is the TCP port for server http requests.  All traffic is redirected to the https port.
		HTTPPort int
		// HTTPSPORT is the TCP port for server https requests.
		HTTPSPort int
		// Tokenizer is used to generate and parse session tokens
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

	// Tokenizer creates and reads tokens from http traffic.
	Tokenizer interface {
		Create(username string, points int) (string, error)
		ReadUsername(tokenString string) (string, error)
	}
)

const (
	// HeaderContentType is used to set the document type header on http responses.
	HeaderContentType = "Content-Type"
	// HeaderCacheControl is used to tell browsers how long to cache http responses.
	HeaderCacheControl = "Cache-Control"
	// HeaderLocation is used to tell browsers to request a different document.
	HeaderLocation = "Location"
	// HeaderAcceptEncoding is specified by the browser to tell the server what types of document encoding it can handle.
	HeaderAcceptEncoding = "Accept-Encoding"
	// HeaderContentEncoding is used to tell browsers how the document is encoded.
	HeaderContentEncoding = "Content-Encoding"
)

// NewServer creates a Server from the Config
func (cfg Config) NewServer(log *log.Logger, tokenizer Tokenizer, userDao UserDao, lobby Lobby, templateFS, staticFS fs.FS) (*Server, error) {
	if err := cfg.validate(log, tokenizer, userDao, lobby); err != nil {
		return nil, fmt.Errorf("creating server: validation: %w", err)
	}
	var gameConfig game.Config
	data := struct {
		Name        string
		ShortName   string
		Description string
		Version     string
		Colors      ColorConfig
		Rules       []string
	}{
		Name:        "selene-bananas",
		ShortName:   "bananas",
		Description: "a tile-based word-forming game",
		Version:     cfg.Version,
		Colors:      cfg.ColorConfig,
		Rules:       gameConfig.Rules(),
	}
	httpAddr := fmt.Sprintf(":%d", cfg.HTTPPort)
	if cfg.HTTPPort <= 0 {
		httpAddr = ""
	}
	httpsAddr := fmt.Sprintf(":%d", cfg.HTTPSPort)
	if cfg.HTTPSPort <= 0 {
		return nil, fmt.Errorf("invalid https port: %v", cfg.HTTPSPort)
	}
	httpServeMux := new(http.ServeMux)
	httpServer := &http.Server{
		Addr:    httpAddr,
		Handler: httpServeMux,
	}
	httpsServeMux := new(http.ServeMux)
	httpsServer := &http.Server{
		Addr:    httpsAddr,
		Handler: httpsServeMux,
	}
	cacheMaxAge := fmt.Sprintf("max-age=%d", cfg.CacheSec)
	templateFileGlobs := []string{
		"html/**/*.html",
		"fa/*.svg",
		"favicon.svg",
		"index.css",
		"*.js",
		"manifest.json",
	}
	template, err := template.ParseFS(templateFS, templateFileGlobs...)
	if err != nil {
		return nil, fmt.Errorf("parsing template: %v", err)
	}
	staticFileSystem := http.FS(staticFS)
	staticFilesHandler := http.FileServer(staticFileSystem)
	s := Server{
		log:         log,
		data:        data,
		tokenizer:   tokenizer,
		userDao:     userDao,
		lobby:       lobby,
		httpServer:  httpServer,
		httpsServer: httpsServer,
		cacheMaxAge: cacheMaxAge,
		template:    template,
		serveStatic: staticFilesHandler,
		Config:      cfg,
	}
	httpServeMux.HandleFunc("/", s.handleHTTP)
	httpsServeMux.HandleFunc("/", s.handleHTTPS)
	return &s, nil
}

// validate ensures the configuration has no errors.
func (cfg Config) validate(log *log.Logger, tokenizer Tokenizer, userDao UserDao, lobby Lobby) error {
	switch {
	case log == nil:
		return fmt.Errorf("log required")
	case tokenizer == nil:
		return fmt.Errorf("tokenizer required")
	case userDao == nil:
		return fmt.Errorf("user dao required")
	case lobby == nil:
		return fmt.Errorf("lobby required")
	case cfg.StopDur <= 0:
		return fmt.Errorf("shop timeout duration required")
	case cfg.CacheSec < 0:
		return fmt.Errorf("non-negative cache time required")
	}
	return nil
}

// Run the server asynchronously until it receives a shutdown signal.
// When the HTTP/HTTPS servers stop, errors are logged to the error channel.
func (s *Server) Run(ctx context.Context) <-chan error {
	errC := make(chan error, 2)
	s.runHTTPServer(ctx, errC)
	s.runHTTPSServer(ctx, errC)
	return errC
}

// runHTTPSServer runs the http server asynchronously, adding the return error to the channel when done.
// The server is only run if the HTTP address is valid.
func (s *Server) runHTTPServer(ctx context.Context, errC chan<- error) {
	if !s.validHTTPAddr() {
		return
	}
	go func() {
		errC <- s.httpServer.ListenAndServe()
	}()
}

// runHTTPSServer runs the https server in regards to the conviguration, adding the return error to the channel when done.
func (s *Server) runHTTPSServer(ctx context.Context, errC chan<- error) {
	ctx, cancelFunc := context.WithCancel(ctx)
	s.lobby.Run(ctx, &s.wg)
	s.httpsServer.RegisterOnShutdown(cancelFunc)
	s.log.Printf("starting https server at at https://127.0.0.1%v", s.httpsServer.Addr)
	go func() {
		switch {
		case s.validHTTPAddr():
			if _, err := tls.LoadX509KeyPair(s.TLSCertFile, s.TLSKeyFile); err != nil {
				errC <- fmt.Errorf("Problem loading tls certificate: %v", err)
				return
			}
			errC <- s.httpsServer.ListenAndServeTLS(s.TLSCertFile, s.TLSKeyFile)
		default:
			if len(s.TLSCertFile) != 0 || len(s.TLSKeyFile) != 0 {
				s.log.Printf("Ignoring TLS_CERT_FILE/TLS_KEY_FILE variables since PORT was specified, using automated certificate management.")
			}
			errC <- s.httpsServer.ListenAndServe()
		}
	}()
}

// Stop asks the server to shutdown and waits for the shutdown to complete.
// An error is returned if the server if the context times out.
func (s *Server) Stop(ctx context.Context) error {
	ctx, cancelFunc := context.WithTimeout(ctx, s.StopDur)
	defer cancelFunc()
	httpsShutdownErr := s.httpsServer.Shutdown(ctx)
	httpShutdownErr := s.httpServer.Shutdown(ctx)
	switch {
	case httpsShutdownErr != nil:
		return httpsShutdownErr
	case httpShutdownErr != nil:
		return httpShutdownErr
	}
	s.wg.Wait()
	return nil
}

// handleHTTPS handles http endpoints.
func (s *Server) handleHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case s.Challenge.IsFor(r.URL.Path):
		if err := s.Challenge.Handle(w, r.URL.Path); err != nil {
			s.log.Printf("serving acme challenge: %v", err)
			s.httpError(w, http.StatusInternalServerError)
		}
	default:
		s.redirectToHTTPS(w, r)
	}
}

// handleHTTPS handles https endpoints.
func (s *Server) handleHTTPS(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.TLS == nil && !s.NoTLSRedirect:
		s.handleHTTP(w, r)
	case r.TLS == nil && s.NoTLSRedirect && !s.hasSecHeader(r):
		s.redirectToHTTPS(w, r)
	case r.Method == "GET":
		s.handleHTTPSGet(w, r)
	case r.Method == "POST":
		s.handleHTTPSPost(w, r)
	default:
		s.httpError(w, http.StatusMethodNotAllowed)
	}
}

// handleHTTPSGet calls handlers for GET endpoints.
func (s *Server) handleHTTPSGet(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/", "/manifest.json", "/serviceWorker.js", "/favicon.svg", "/network_check.html":
		s.handleFile(w, r, s.serveTemplate(r.URL.Path))
	case "/wasm_exec.js", "/main.wasm", "/robots.txt", "/favicon.png", "/LICENSE":
		s.handleFile(w, r, s.serveStatic)
	case "/lobby":
		s.handleUserLobby(w, r)
	case "/monitor":
		s.handleMonitor(w, r)
	default:
		s.httpError(w, http.StatusNotFound)
	}
}

// handleHTTPSPost checks authentication and calls handlers for POST endpoints.
func (s *Server) handleHTTPSPost(w http.ResponseWriter, r *http.Request) {
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
	case "/ping":
		// NOOP
	default:
		s.httpError(w, http.StatusNotFound)
	}
}

// handleFile wraps the handling of the file, add cache-control header and gzip compression, if possible.
func (s *Server) handleFile(w http.ResponseWriter, r *http.Request, h http.Handler) {
	switch r.URL.Path {
	default:
		if r.URL.Query().Get("v") != s.Version {
			url := r.URL
			q := url.Query()
			q.Set("v", s.Version)
			url.RawQuery = q.Encode()
			w.Header().Set(HeaderLocation, url.String())
			w.WriteHeader(http.StatusMovedPermanently)
			return
		}
		fallthrough
	case "/favicon.svg", "/favicon.png":
		w.Header().Set(HeaderCacheControl, s.cacheMaxAge)
	case "/":
		w.Header().Set(HeaderCacheControl, "no-store")
	}
	if strings.Contains(r.Header.Get(HeaderAcceptEncoding), "gzip") {
		w2 := gzip.NewWriter(w)
		defer w2.Close()
		w = wrappedResponseWriter{
			Writer:         w2,
			ResponseWriter: w,
		}
		w.Header().Set(HeaderContentEncoding, "gzip")
	}
	h.ServeHTTP(w, r)
}

// serveTemplate servers the file from the data-driven template.  The name is assumed to have a leading slash that is ignored.
func (s *Server) serveTemplate(name string) http.HandlerFunc {
	name = name[1:]
	if len(name) == 0 {
		name = "index.html"
	}
	return func(w http.ResponseWriter, r *http.Request) {
		s.addMimeType(name, w)
		if err := s.template.ExecuteTemplate(w, name, s.data); err != nil {
			err = fmt.Errorf("rendering template: %v", err)
			s.handleError(w, err)
			return
		}
	}
}

// validHTTPAddr determines if the HTTP address is valid.
// If the HTTP address is valid, the HTTP server should be started to redirect to HTTPS and handle certificate creation.
func (s *Server) validHTTPAddr() bool {
	return len(s.httpServer.Addr) > 0
}

// redirectToHTTPS redirects the page to https.
func (s *Server) redirectToHTTPS(w http.ResponseWriter, r *http.Request) {
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
	if s.httpsServer.Addr != ":443" && !s.NoTLSRedirect {
		host = host + s.httpsServer.Addr
	}
	httpsURI := "https://" + host + r.URL.Path
	http.Redirect(w, r, httpsURI, http.StatusTemporaryRedirect)
}

// checkTokenUsername ensures the username in the authorization header matches that in the username form value.
func (s *Server) checkTokenUsername(r *http.Request) error {
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

// handleError logs and writes the error as an internal server error (500).
func (s *Server) handleError(w http.ResponseWriter, err error) {
	s.log.Printf("server error: %v", err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

// httpError writes the error status code.
func (*Server) httpError(w http.ResponseWriter, statusCode int) {
	http.Error(w, http.StatusText(statusCode), statusCode)
}

// hasSecHeader returns true if thhe request has any header starting with "Sec-".
func (*Server) hasSecHeader(r *http.Request) bool {
	for header := range r.Header {
		if strings.HasPrefix(header, "Sec-") {
			return true
		}
	}
	return false
}

// addMimeType adds the applicable mime type to the response.
func (*Server) addMimeType(fileName string, w http.ResponseWriter) {
	extension := filepath.Ext(fileName)
	mimeType := mime.TypeByExtension(extension)
	w.Header().Set(HeaderContentType, mimeType)
}

// wrappedResponseWriter wraps response writing with another writer.
type wrappedResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

// Write delegates the write to the wrapped writer.
func (wrw wrappedResponseWriter) Write(p []byte) (n int, err error) {
	return wrw.Writer.Write(p)
}

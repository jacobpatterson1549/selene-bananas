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
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

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
		challenge   Challenge
		httpServer  *http.Server
		httpsServer *http.Server
		cacheMaxAge string
		template    *template.Template
		serveStatic http.Handler
		monitor     http.Handler
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

	// Challenge is used to fulfill authentication checks to get a certificate.
	Challenge interface {
		IsFor(path string) bool
		Handle(w io.Writer, path string) error
	}

	// Parameters contains the interfaces needed to create a new server
	Parameters struct {
		Log *log.Logger
		Tokenizer
		UserDao
		Lobby
		Challenge
		StaticFS   fs.FS
		TemplateFS fs.FS
		sqlFiles   []fs.File
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
	// rootTemplateName is the name of the template for the root of the site
	rootTemplateName = "index.html"
)

// NewServer creates a Server from the Config
func (cfg Config) NewServer(p Parameters) (*Server, error) {
	cfg.Version = strings.TrimSpace(cfg.Version) // TODO: test this
	if err := cfg.validate(p); err != nil {
		return nil, fmt.Errorf("creating server: validation: %w", err)
	}
	template, err := p.parseTemplate()
	if err != nil {
		return nil, err
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
	staticFileSystem := http.FS(p.StaticFS)
	staticFilesHandler := http.FileServer(staticFileSystem)
	s := Server{
		log:         p.Log,
		data:        data,
		tokenizer:   p.Tokenizer,
		userDao:     p.UserDao,
		lobby:       p.Lobby,
		challenge:   p.Challenge,
		httpServer:  httpServer,
		httpsServer: httpsServer,
		cacheMaxAge: cacheMaxAge,
		template:    template,
		serveStatic: staticFilesHandler,
		Config:      cfg,
	}
	s.monitor = runtimeMonitor{
		hasTLS: s.validHTTPAddr(),
	}
	httpServeMux.HandleFunc("/", s.handleHTTP)
	httpsServeMux.HandleFunc("/", s.handleHTTPS)
	return &s, nil
}

// validate ensures the configuration and parameters have no errors.
func (cfg Config) validate(p Parameters) error {
	if err := p.validate(); err != nil {
		return err
	}
	switch {
	case cfg.StopDur <= 0:
		return fmt.Errorf("stop timeout duration required")
	case cfg.CacheSec < 0:
		return fmt.Errorf("nonnegative cache seconds required")
	case cfg.HTTPSPort <= 0:
		return fmt.Errorf("positive https port required")
	case len(cfg.Version) == 0:
		return fmt.Errorf("version required")
	}
	for i, r := range cfg.Version {
		if !unicode.In(r, unicode.Letter, unicode.Digit) {
			return fmt.Errorf("only letters and digits are allowed in version: invalid rune at index %v of '%v': '%v'", i, cfg.Version, string(r))
		}
	}
	return nil
}

// validate ensures that all of the parameters are present.
func (p Parameters) validate() error {
	switch {
	case p.Log == nil:
		return fmt.Errorf("log required")
	case p.Tokenizer == nil:
		return fmt.Errorf("tokenizer required")
	case p.UserDao == nil:
		return fmt.Errorf("user dao required")
	case p.Lobby == nil:
		return fmt.Errorf("lobby required")
	case p.Challenge == nil:
		return fmt.Errorf("challenge required")
	case p.StaticFS == nil:
		return fmt.Errorf("static file system required")
	case p.TemplateFS == nil:
		return fmt.Errorf("template file system required")
	}
	return nil
}

// parseTemplate parses the whole template file system to create a template.
func (p Parameters) parseTemplate() (*template.Template, error) {
	t, err := template.ParseFS(p.TemplateFS, "*")
	if err != nil {
		return nil, fmt.Errorf("parsing template file system: %v", err)
	}
	return t, nil
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
	case s.challenge.IsFor(r.URL.Path):
		if err := s.challenge.Handle(w, r.URL.Path); err != nil {
			err := fmt.Errorf("serving acme challenge: %v", err)
			s.writeInternalError(w, err)
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
	case r.TLS == nil && s.NoTLSRedirect && !hasSecHeader(r):
		s.redirectToHTTPS(w, r)
	case r.Method == "GET":
		s.handleGet(w, r)
	case r.Method == "POST":
		s.handlePost(w, r)
	default:
		httpError(w, http.StatusMethodNotAllowed)
	}
}

// handleGet calls handlers for GET endpoints.
func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/", "/manifest.json", "/serviceWorker.js", "/favicon.svg", "/network_check.html":
		s.handleFile(w, r, s.serveTemplate(r.URL.Path[1:]))
	case "/wasm_exec.js", "/main.wasm", "/robots.txt", "/favicon.png", "/LICENSE":
		s.handleFile(w, r, s.serveStatic)
	case "/lobby":
		s.handleUserLobby(w, r)
	case "/monitor":
		s.monitor.ServeHTTP(w, r)
	default:
		httpError(w, http.StatusNotFound)
	}
}

// handlePost checks authentication and calls handlers for POST endpoints.
func (s *Server) handlePost(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user_create", "/user_login":
		// [unauthenticated]
	default:
		if err := s.checkTokenUsername(r); err != nil {
			s.log.Print(err)
			httpError(w, http.StatusForbidden)
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
		httpError(w, http.StatusNotFound)
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
		w.Header().Add(HeaderContentEncoding, "gzip")
	}
	addMimeType(r.URL.Path, w)
	h.ServeHTTP(w, r)
}

// serveTemplate servers the file from the data-driven template.  The name is assumed to have a leading slash that is ignored.
func (s *Server) serveTemplate(name string) http.HandlerFunc {
	if len(name) == 0 {
		name = rootTemplateName
	}
	return func(w http.ResponseWriter, r *http.Request) {
		addMimeType(name, w)
		if err := s.template.ExecuteTemplate(w, name, s.data); err != nil {
			err = fmt.Errorf("rendering template: %v", err)
			s.writeInternalError(w, err)
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
	// derived from net.SplitHostPort, but does not throw error :
	lastColonIndex := strings.LastIndex(host, ":")
	if lastColonIndex >= 0 {
		host = host[:lastColonIndex]
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

// writeInternalError logs and writes the error as an internal server error (500).
func (s *Server) writeInternalError(w http.ResponseWriter, err error) {
	s.log.Printf("server error: %v", err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

// httpError writes the error status code.
func httpError(w http.ResponseWriter, statusCode int) {
	http.Error(w, http.StatusText(statusCode), statusCode)
}

// hasSecHeader returns true if thhe request has any header starting with "Sec-".
func hasSecHeader(r *http.Request) bool {
	for header := range r.Header {
		if strings.HasPrefix(header, "Sec-") {
			return true
		}
	}
	return false
}

// addMimeType adds the applicable mime type to the response.
func addMimeType(fileName string, w http.ResponseWriter) {
	switch {
	case fileName == "/":
		fileName = ".html" // assume html for page root
	case !strings.Contains(fileName, "."):
		fileName = ".txt"
	}
	extension := filepath.Ext(fileName)
	mimeType := mime.TypeByExtension(extension)
	w.Header().Add(HeaderContentType, mimeType)
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

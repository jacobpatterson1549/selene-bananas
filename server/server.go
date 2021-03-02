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
		// TLSCertPEM is the public HTTPS TLS certificate file data.
		TLSCertPEM string
		// TLSKeyPEM is the private HTTPS TLS key file data.
		TLSKeyPEM string
		// Challenge is used to create ACME certificate.
		Challenge
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

	// Challenge token and key used to get a TLS certificate using the ACME HTTP-01.
	Challenge struct {
		Token string
		Key   string
	}

	// Parameters contains the interfaces needed to create a new server
	Parameters struct {
		Log *log.Logger
		Tokenizer
		UserDao
		Lobby
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
	// rootTemplatePath is the name of the template for the root of the site
	rootTemplatePath = "/index.html"
	// acmeHeader is the path of the endpoint to serve the challenge at.
	acmeHeader = "/.well-known/acme-challenge/"
)

// NewServer creates a Server from the Config
func (cfg Config) NewServer(p Parameters) (*Server, error) {
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
	httpServer := &http.Server{
		Addr: httpAddr,
	}
	httpsServer := &http.Server{
		Addr: httpsAddr,
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
	s.httpServer.Handler = s.httpHandler()
	s.httpsServer.Handler = s.httpsHandler()
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
		errC <- s.serveHTTPS()
	}()
}

// serveHTTPS is closely derived from https://golang.org/src/net/http/server.go to allow key bytes rather than files
func (s *Server) serveHTTPS() error {
	ln, err := net.Listen("tcp", s.httpsServer.Addr)
	defer ln.Close()
	if err != nil {
		return err
	}
	switch {
	case s.validHTTPAddr():
		ln, err = s.tlsListener(ln)
		if err != nil {
			return err
		}
	default:
		if len(s.TLSCertPEM) != 0 || len(s.TLSKeyPEM) != 0 {
			s.log.Printf("Ignoring certificate since PORT was specified, using automated certificate management.")
		}
	}
	return s.httpsServer.Serve(ln) // BLOCKING
}

// tlsListener is derived from https://golang.org/src/net/http/server.go to allow key bytes rather than files
func (s *Server) tlsListener(l net.Listener) (net.Listener, error) {
	tlsConfig := s.httpServer.TLSConfig
	switch {
	case tlsConfig == nil:
		tlsConfig = &tls.Config{}
	default:
		tlsConfig = tlsConfig.Clone()
	}
	tlsConfig.NextProtos = append(tlsConfig.NextProtos, "http/1.1")
	var err error
	tlsConfig.Certificates = make([]tls.Certificate, 1)
	tlsConfig.Certificates[0], err = tls.X509KeyPair([]byte(s.TLSCertPEM), []byte(s.TLSKeyPEM))
	if err != nil {
		return nil, err
	}
	tlsListener := tls.NewListener(l, tlsConfig)
	return tlsListener, nil
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

// handleHTTPS creates a handler for HTTP endpoints.
func (s *Server) httpHandler() http.Handler {
	httpMux := http.NewServeMux()
	httpMux.HandleFunc(acmeHeader, http.HandlerFunc(s.handleAcmeChallenge))
	httpMux.Handle("/", http.HandlerFunc(s.redirectToHTTPS))
	return httpMux
}

// httpsHandler creates a handler for HTTPS endpoints.
// Non-TLS requests are redirected to HTTPS.  GET and POST requests are handled by more specific handlers.
func (s *Server) httpsHandler() http.HandlerFunc {
	getHandler := s.getHandler()
	postHandler := s.postHandler()
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.TLS == nil && !s.NoTLSRedirect:
			s.httpServer.Handler.ServeHTTP(w, r)
		case r.TLS == nil && s.NoTLSRedirect && !hasSecHeader(r):
			s.redirectToHTTPS(w, r)
		case r.Method == "GET":
			getHandler.ServeHTTP(w, r)
		case r.Method == "POST":
			postHandler.ServeHTTP(w, r)
		default:
			httpError(w, http.StatusMethodNotAllowed)
		}
	}
}

// getHandler forwards calls to various endpoints.
func (s *Server) getHandler() http.Handler {
	getMux := http.NewServeMux()
	templatePatterns := []string{rootTemplatePath, "/manifest.json", "/serviceWorker.js", "/favicon.svg", "/network_check.html"}
	staticPatterns := []string{"/wasm_exec.js", "/main.wasm", "/robots.txt", "/favicon.png", "/LICENSE"}
	templateHandler := s.fileHandler(http.HandlerFunc(s.serveTemplate))
	staticHandler := s.fileHandler(s.serveStatic)
	for _, p := range templatePatterns {
		getMux.Handle(p, templateHandler)
	}
	for _, p := range staticPatterns {
		getMux.Handle(p, staticHandler)
	}
	getMux.Handle("/lobby", http.HandlerFunc(s.handleUserLobby))
	getMux.Handle("/monitor", s.monitor)
	return s.rootHandler(getMux)
}

// postHandler checks authentication and calls handlers for POST endpoints.
func (s *Server) postHandler() http.Handler {
	postMux := http.NewServeMux()
	noopHandler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// NOOP
	})
	postMux.Handle("/user_create", http.HandlerFunc(s.handleUserCreate))
	postMux.Handle("/user_login", http.HandlerFunc(s.handleUserLogin))
	postMux.Handle("/user_update_password", http.HandlerFunc(s.handleUserUpdatePassword))
	postMux.Handle("/user_delete", http.HandlerFunc(s.handleUserDelete))
	postMux.Handle("/ping", noopHandler) // NOOP
	return s.authHandler(postMux)
}

// rootHandler maps requests for / to /index.html.
func (*Server) rootHandler(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			r.URL.Path = rootTemplatePath
		}
		h.ServeHTTP(w, r)
	}
}

// authHandler checks the token username of the request before running the child handler
func (s *Server) authHandler(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		h.ServeHTTP(w, r)
	}
}

// handleAcmeChallenge writes the challenge to the response.
// Writes the concatenation of the token, a period, and the key.
func (s *Server) handleAcmeChallenge(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path[len(acmeHeader):] != s.Challenge.Token {
		err := fmt.Errorf("path '%v' is not for challenge", path)
		s.writeInternalError(w, err)
	}
	data := s.Challenge.Token + "." + s.Challenge.Key
	w.Write([]byte(data))
}

// fileHandler wraps the handling of the file, add cache-control header and gzip compression, if possible.
func (s *Server) fileHandler(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		case rootTemplatePath:
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
}

// serveTemplate servers the file from the data-driven template.  The name is assumed to have a leading slash that is ignored.
func (s *Server) serveTemplate(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Path[1:] // ignore leading slash
	addMimeType(name, w)
	if err := s.template.ExecuteTemplate(w, name, s.data); err != nil {
		err = fmt.Errorf("rendering template: %v", err)
		s.writeInternalError(w, err)
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

// addMimeType adds the applicable mime type to the response.  Files without extensions are assumed to be text
func addMimeType(fileName string, w http.ResponseWriter) {
	if !strings.Contains(fileName, ".") {
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

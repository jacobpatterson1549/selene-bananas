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
		lobby       Lobby
		HTTPServer  *http.Server
		HTTPSServer *http.Server
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
	httpAddr := fmt.Sprintf(":%d", cfg.HTTPPort)
	httpsAddr := fmt.Sprintf(":%d", cfg.HTTPSPort)
	httpsRedirectHandler := cfg.httpsRedirectHandler()
	httpHandler := cfg.httpHandler(httpsRedirectHandler)
	httpsHandler := cfg.httpsHandler(httpHandler, httpsRedirectHandler, p, template)
	s := Server{
		log:   p.Log,
		lobby: p.Lobby,
		HTTPServer: &http.Server{
			Addr:    httpAddr,
			Handler: httpHandler,
		},
		HTTPSServer: &http.Server{
			Addr:    httpsAddr,
			Handler: httpsHandler,
		},
		Config: cfg,
	}
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

// date is a structure of variables to insert into templates.
func (cfg Config) data() interface{} {
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
	return data
}

// validHTTPAddr determines if the HTTP address is valid.
// The HTTP address is valid if and only if the HTTP port is positive
// If the HTTP address is valid, the HTTP server should be started to redirect to HTTPS and handle certificate creation.
func (cfg Config) validHTTPAddr() bool {
	return cfg.HTTPPort > 0
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
	go s.serveTCP(s.HTTPServer, errC, false)
}

// runHTTPSServer runs the https server in regards to the conviguration, adding the return error to the channel when done.
func (s *Server) runHTTPSServer(ctx context.Context, errC chan<- error) {
	ctx, cancelFunc := context.WithCancel(ctx)
	s.lobby.Run(ctx, &s.wg)
	s.HTTPSServer.RegisterOnShutdown(cancelFunc)
	s.logServerStart()
	go s.serveTCP(s.HTTPSServer, errC, true)
}

func (s *Server) logServerStart() {
	var serverStartInfo string
	switch {
	case s.validHTTPAddr():
		serverStartInfo = "starting server at at https://127.0.0.1%v"
	default:
		serverStartInfo = "starting server at at http://127.0.0.1%v"
	}
	s.log.Printf(serverStartInfo, s.HTTPSServer.Addr)
}

// serveTCP is closely derived from https://golang.org/src/net/http/server.go to allow key bytes rather than files
func (s *Server) serveTCP(svr *http.Server, errC chan<- error, tls bool) (err error) {
	defer func() { errC <- err }()
	ln, err := net.Listen("tcp", svr.Addr)
	if err != nil {
		return err
	}
	defer ln.Close()
	switch {
	case !tls:
		// NOOP
	case s.validHTTPAddr():
		ln, err = s.tlsListener(ln)
		if err != nil {
			return err
		}
	case len(s.TLSCertPEM) != 0, len(s.TLSKeyPEM) != 0:
		s.log.Printf("Ignoring certificate since PORT was specified, using automated certificate management.")
	}
	err = svr.Serve(ln) // BLOCKING
	return
}

// tlsListener is derived from https://golang.org/src/net/http/server.go to allow key bytes rather than files
func (s *Server) tlsListener(l net.Listener) (net.Listener, error) {
	certificate, err := tls.X509KeyPair([]byte(s.TLSCertPEM), []byte(s.TLSKeyPEM))
	if err != nil {
		return nil, err
	}
	tlsCfg := &tls.Config{}
	tlsCfg.NextProtos = []string{"http/1.1"}
	tlsCfg.Certificates = []tls.Certificate{certificate}
	tlsListener := tls.NewListener(l, tlsCfg)
	return tlsListener, nil
}

// Stop asks the server to shutdown and waits for the shutdown to complete.
// An error is returned if the server if the context times out.
func (s *Server) Stop(ctx context.Context) error {
	ctx, cancelFunc := context.WithTimeout(ctx, s.StopDur)
	defer cancelFunc()
	httpsShutdownErr := s.HTTPSServer.Shutdown(ctx)
	httpShutdownErr := s.HTTPServer.Shutdown(ctx)
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
func (cfg Config) httpHandler(httpsRedirectHandler http.Handler) http.Handler {
	httpMux := http.NewServeMux()
	httpMux.HandleFunc(acmeHeader, http.HandlerFunc(acmeChallengeHandler(cfg.Challenge)))
	httpMux.Handle("/", httpsRedirectHandler)
	return httpMux
}

// httpsHandler creates a handler for HTTPS endpoints.
// Non-TLS requests are redirected to HTTPS.  GET and POST requests are handled by more specific handlers.
func (cfg Config) httpsHandler(httpHandler, httpsRedirectHandler http.Handler, p Parameters, template *template.Template) http.HandlerFunc {
	getHandler := p.getHandler(cfg, template)
	postHandler := p.postHandler()
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.TLS == nil && !cfg.NoTLSRedirect:
			httpHandler.ServeHTTP(w, r)
		case r.TLS == nil && cfg.NoTLSRedirect && !hasSecHeader(r):
			httpsRedirectHandler.ServeHTTP(w, r)
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
func (p Parameters) getHandler(cfg Config, template *template.Template) http.Handler {
	data := cfg.data()
	cacheMaxAge := fmt.Sprintf("max-age=%d", cfg.CacheSec)
	staticFileHandler := http.FileServer(http.FS(p.StaticFS))
	templateHandler := fileHandler(http.HandlerFunc(templateHandler(template, data)), cacheMaxAge)
	staticHandler := fileHandler(staticFileHandler, cacheMaxAge)
	monitor := runtimeMonitor{
		hasTLS: cfg.validHTTPAddr(),
	}
	templatePatterns := []string{rootTemplatePath, "/manifest.json", "/serviceWorker.js", "/favicon.svg", "/network_check.html"}
	staticPatterns := []string{"/wasm_exec.js", "/main.wasm", "/robots.txt", "/favicon.png", "/favicon.ico", "/LICENSE"}

	getMux := http.NewServeMux()
	for _, p := range templatePatterns {
		getMux.Handle(p, templateHandler)
	}
	for _, p := range staticPatterns {
		getMux.Handle(p, staticHandler)
	}
	getMux.Handle("/lobby", http.HandlerFunc(userLobbyConnectHandler(p.Tokenizer, p.Lobby, p.Log)))
	getMux.Handle("/monitor", monitor)
	return rootHandler(getMux)
}

// postHandler checks authentication and calls handlers for POST endpoints.
func (p Parameters) postHandler() http.Handler {
	postMux := http.NewServeMux()
	postMux.Handle("/user_create", http.HandlerFunc(userCreateHandler(p.UserDao, p.Log)))
	postMux.Handle("/user_login", http.HandlerFunc(userLoginHandler(p.UserDao, p.Tokenizer, p.Log)))
	postMux.Handle("/user_update_password", http.HandlerFunc(userUpdatePasswordHandler(p.UserDao, p.Lobby, p.Log)))
	postMux.Handle("/user_delete", http.HandlerFunc(userDeleteHandler(p.UserDao, p.Lobby, p.Log)))
	postMux.Handle("/ping", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// NOOP
	}))
	return authHandler(postMux, p.Tokenizer, p.Log)
}

// rootHandler maps requests for / to /index.html.
func rootHandler(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			r.URL.Path = rootTemplatePath
		}
		h.ServeHTTP(w, r)
	}
}

// authHandler checks the token username of the request before running the child handler
func authHandler(h http.Handler, tokenizer Tokenizer, log *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user_create", "/user_login":
			// [unauthenticated]
		default:
			if err := checkTokenUsername(r, tokenizer); err != nil {
				log.Print(err)
				httpError(w, http.StatusForbidden)
				return
			}
		}
		h.ServeHTTP(w, r)
	}
}

// acmeChallengeHandler writes the challenge to the response.
// Writes the concatenation of the token, a period, and the key.
func acmeChallengeHandler(challenge Challenge) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path[len(acmeHeader):] != challenge.Token {
			http.NotFound(w, r)
			return
		}
		data := challenge.Token + "." + challenge.Key
		w.Write([]byte(data))
	}
}

// fileHandler wraps the handling of the file, add cache-control header and gzip compression, if possible.
func fileHandler(h http.Handler, cacheMaxAge string) http.HandlerFunc {
	cacheControl := func(r *http.Request) string {
		switch r.URL.Path {
		case rootTemplatePath:
			return "no-store"
		}
		return cacheMaxAge
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get(HeaderAcceptEncoding), "gzip") {
			w2 := gzip.NewWriter(w)
			defer w2.Close()
			w = wrappedResponseWriter{
				Writer:         w2,
				ResponseWriter: w,
			}
			w.Header().Add(HeaderContentEncoding, "gzip")
		}
		w.Header().Set(HeaderCacheControl, cacheControl(r))
		addMimeType(r.URL.Path, w)
		h.ServeHTTP(w, r)
	}
}

// templateHandler servers the file from the data-driven template.  The name is assumed to have a leading slash that is ignored.
func templateHandler(template *template.Template, data interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Path[1:] // ignore leading slash
		template.ExecuteTemplate(w, name, data)
	}
}

// httpsRedirectHandler redirects the request to https.
func (cfg Config) httpsRedirectHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		// derived from net.SplitHostPort, but does not throw error :
		lastColonIndex := strings.LastIndex(host, ":")
		if lastColonIndex >= 0 {
			host = host[:lastColonIndex]
		}
		if cfg.HTTPSPort != 443 && !cfg.NoTLSRedirect {
			host += fmt.Sprintf(":%d", cfg.HTTPSPort)
		}
		httpsURI := "https://" + host + r.URL.Path
		http.Redirect(w, r, httpsURI, http.StatusTemporaryRedirect)
	}
}

// checkTokenUsername ensures the username in the authorization header matches that in the username form value.
func checkTokenUsername(r *http.Request, tokenizer Tokenizer) error {
	authorization := r.Header.Get("Authorization")
	if len(authorization) < 7 || authorization[:7] != "Bearer " {
		return fmt.Errorf("invalid authorization header: %v", authorization)
	}
	tokenString := authorization[7:]
	tokenUsername, err := tokenizer.ReadUsername(tokenString)
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
func writeInternalError(err error, log *log.Logger, w http.ResponseWriter) {
	log.Printf("server error: %v", err)
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

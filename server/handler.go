package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/jacobpatterson1549/selene-bananas/db/user"
	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/server/log"
	"github.com/jacobpatterson1549/selene-bananas/server/oauth2"
)

type (
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

	// Parameters contains the interfaces needed to create a new server
	Parameters struct {
		log.Logger
		Tokenizer
		UserDao
		Lobby
		StaticFS       fs.FS
		TemplateFS     fs.FS
		GoogleEndpoint *oauth2.Endpoint
	}

	// Challenge token and key used to get a TLS certificate using the ACME HTTP-01.
	Challenge struct {
		Token string
		Key   string
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

	contextKey int

	templateData struct {
		Name           string
		ShortName      string
		Description    string
		Version        string
		Colors         ColorConfig
		Rules          []string
		HasUserDB      bool
		CanGoogleLogin bool
		JWT            string
		AccessToken    string
		JWTUser        user.User
	}
)

const (
	// HeaderContentType is used to set the document type header on http responses.
	HeaderContentType = "Content-Type"
	// HeaderCacheControl is used to tell browsers how long to cache http responses.
	HeaderCacheControl = "Cache-Control"
	// HeaderStrictTransportSecurity is used to tell browsers the sit should only be accessed using HTTPS.
	HeaderStrictTransportSecurity = "Strict-Transport-Security"
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

const (
	usernameContextKey contextKey = iota + 1
	isOauth2ContextKey
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
	monitor := runtimeMonitor{
		hasTLS: cfg.validHTTPAddr(),
	}
	httpAddr := fmt.Sprintf(":%d", cfg.HTTPPort)
	httpsAddr := fmt.Sprintf(":%d", cfg.HTTPSPort)
	httpsRedirectHandler := httpsRedirectHandler(cfg.HTTPSPort)
	httpHandler := cfg.httpHandler(httpsRedirectHandler)
	httpsHandler := cfg.httpsHandler(httpHandler, httpsRedirectHandler, p, template, monitor)
	s := Server{
		log:   p.Logger,
		lobby: p.Lobby,
		HTTPServer: &http.Server{
			Addr:         httpAddr,
			Handler:      httpHandler,
			ReadTimeout:  60 * time.Second,
			WriteTimeout: 60 * time.Second,
		},
		HTTPSServer: &http.Server{
			Addr:         httpsAddr,
			Handler:      httpsHandler,
			ReadTimeout:  60 * time.Second,
			WriteTimeout: 60 * time.Second,
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

// init configures the structure of variables to insert into templates.
func (cfg Config) newTemplateData() *templateData {
	var defaultGameCfg game.Config
	rules := defaultGameCfg.Rules()
	data := templateData{
		Name:        "selene-bananas",
		ShortName:   "bananas",
		Description: "a tile-based word-forming game",
		Version:     cfg.Version,
		Colors:      cfg.ColorConfig,
		Rules:       rules,
	}
	return &data
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
	case p.Logger == nil:
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

// handleHTTPS creates a handler for HTTP endpoints.
func (cfg Config) httpHandler(httpsRedirectHandler http.Handler) http.Handler {
	httpMux := http.NewServeMux()
	httpMux.HandleFunc(acmeHeader, http.HandlerFunc(acmeChallengeHandler(cfg.Challenge)))
	httpMux.Handle("/", httpsRedirectHandler)
	return httpMux
}

// httpsHandler creates a handler for HTTPS endpoints.
// Non-TLS requests are redirected to HTTPS.  GET and POST requests are handled by more specific handlers.
func (cfg Config) httpsHandler(httpHandler, httpsRedirectHandler http.Handler, p Parameters, template *template.Template, monitor http.Handler) http.HandlerFunc {
	getHandler := p.getHandler(cfg, template, monitor)
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
func (p Parameters) getHandler(cfg Config, template *template.Template, monitor http.Handler) http.Handler {
	cacheMaxAge := fmt.Sprintf("max-age=%d", cfg.CacheSec)

	data := cfg.newTemplateData()
	_, noUserDB := p.UserDao.Backend().(user.NoDatabaseBackend)
	data.HasUserDB = !noUserDB
	data.CanGoogleLogin = p.GoogleEndpoint != nil

	templateFileHandler := templateHandler(template, *data, p.Logger)
	staticFileHandler := http.FileServer(http.FS(p.StaticFS))
	templateHandler := fileHandler(http.HandlerFunc(templateFileHandler), cacheMaxAge)
	staticHandler := fileHandler(staticFileHandler, cacheMaxAge)
	templatePatterns := []string{rootTemplatePath, "/manifest.json", "/serviceWorker.js", "/favicon.svg", "/network_check.html"}
	staticPatterns := []string{"/wasm_exec.js", "/selene-bananas.wasm", "/robots.txt", "/favicon.png", "/favicon.ico", "/LICENSE"}

	getMux := http.NewServeMux()
	for _, p := range templatePatterns {
		getMux.Handle(p, templateHandler)
	}
	for _, p := range staticPatterns {
		getMux.Handle(p, staticHandler)
	}
	getMux.Handle("/lobby", http.HandlerFunc(userLobbyConnectHandler(p.Lobby, p.Tokenizer, p.Logger)))
	getMux.Handle("/monitor", monitor)
	if p.GoogleEndpoint != nil {
		jwtHandler := oauth2JWTTemplateHandler(template, *data, p.Logger)
		getMux.Handle(oauth2.GoogleLoginURL, p.GoogleEndpoint.HandleLogin())
		getMux.Handle(oauth2.GoogleCallbackURL, p.GoogleEndpoint.HandleCallback(p.UserDao, p.Tokenizer, jwtHandler))
	}
	return rootHandler(getMux)
}

// postHandler checks authentication and calls handlers for POST endpoints.
func (p Parameters) postHandler() http.Handler {
	postMux := http.NewServeMux()
	postMux.Handle("/user_create", http.HandlerFunc(userCreateHandler(p.UserDao, p.Logger)))
	postMux.Handle("/user_login", http.HandlerFunc(userLoginHandler(p.UserDao, p.Tokenizer, p.Logger)))
	postMux.Handle("/user_update_password", http.HandlerFunc(userUpdatePasswordHandler(p.UserDao, p.Lobby, p.Logger)))
	postMux.Handle("/user_delete", http.HandlerFunc(userDeleteHandler(p.UserDao, p.GoogleEndpoint, p.Lobby, p.Logger)))
	postMux.Handle("/ping", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// NOOP
	}))
	return authHandler(postMux, p.Tokenizer, p.Logger)
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
func authHandler(h http.Handler, tokenizer Tokenizer, log log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user_create", "/user_login":
			// [unauthenticated]
		default:
			authorization := r.Header.Get("Authorization")
			r = withAuthorization(w, r, authorization, tokenizer, log)
		}
		h.ServeHTTP(w, r)
	}
}

// setAuthorization loads the token from the authorization header into the request context.
func withAuthorization(w http.ResponseWriter, r *http.Request, authorization string, tokenizer Tokenizer, log log.Logger) *http.Request {
	username, isOauth2, err := getToken(authorization, tokenizer)
	if err != nil {
		log.Printf(err.Error())
		httpError(w, http.StatusForbidden)
		return r
	}
	ctx := r.Context()
	ctx = context.WithValue(ctx, usernameContextKey, username)
	ctx = context.WithValue(ctx, isOauth2ContextKey, isOauth2)
	return r.WithContext(ctx)
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
		w.Header().Set(HeaderStrictTransportSecurity, cacheMaxAge)
		addMimeType(r.URL.Path, w)
		h.ServeHTTP(w, r)
	}
}

// templateHandler servers the file from the data-driven template.  The name is assumed to have a leading slash that is ignored.
// Templates are written a buffer to ensure they execute correctly before they are written to the response
func templateHandler(template *template.Template, data templateData, log log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Path[1:] // ignore leading slash
		var buf bytes.Buffer
		if err := template.ExecuteTemplate(&buf, name, data); err != nil {
			err = fmt.Errorf("rendering template: %v", err)
			writeInternalError(err, log, w)
		}
		w.Write(buf.Bytes())
	}
}

// oauth2JWTTemplateHandler adds the jwt token to the template data before handling
func oauth2JWTTemplateHandler(template *template.Template, data templateData, log log.Logger) func(jwt, accessToken string, u user.User) http.HandlerFunc {
	return func(jwt, accessToken string, u user.User) http.HandlerFunc {
		data.JWT = jwt
		data.AccessToken = accessToken
		data.JWTUser = u
		h := templateHandler(template, data, log)
		return func(w http.ResponseWriter, r *http.Request) {
			r.URL.Path = rootTemplatePath
			h.ServeHTTP(w, r)
		}
	}
}

// httpsRedirectHandler redirects the request to https.
func httpsRedirectHandler(httpsPort int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		// derived from net.SplitHostPort, but does not throw error :
		lastColonIndex := strings.LastIndex(host, ":")
		if lastColonIndex >= 0 {
			host = host[:lastColonIndex]
		}
		if httpsPort != 443 {
			host += fmt.Sprintf(":%d", httpsPort)
		}
		httpsURI := "https://" + host + r.URL.Path
		http.Redirect(w, r, httpsURI, http.StatusMovedPermanently)
	}
}

// getToken retrieves the username and isOauth2 in the authorization header.
func getToken(authorization string, tokenizer Tokenizer) (username string, isOauth2 bool, err error) {
	if len(authorization) < 7 || authorization[:7] != "Bearer " {
		err = fmt.Errorf("invalid authorization header: %v", authorization)
		return
	}
	tokenString := authorization[7:]
	username, isOauth2, err = tokenizer.Read(tokenString)
	if err != nil {
		err = fmt.Errorf("reading token info: %w", err)
		return
	}
	return
}

// writeInternalError logs and writes the error as an internal server error (500).
func writeInternalError(err error, log log.Logger, w http.ResponseWriter) {
	log.Printf("server error: %v", err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

// httpError writes the error status code.
func httpError(w http.ResponseWriter, statusCode int) {
	http.Error(w, http.StatusText(statusCode), statusCode)
}

// hasSecHeader returns true if the request has any header starting with "Sec-".
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

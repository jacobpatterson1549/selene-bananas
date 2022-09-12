package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/db/user"
	"github.com/jacobpatterson1549/selene-bananas/server/log/logtest"
)

const indexHTML = "index.html"

func TestNewServer(t *testing.T) {
	testLog := logtest.DiscardLogger
	var tokenizer mockTokenizer
	userDao := mockUserDao{
		backendFunc: func() user.Backend {
			return nil
		},
	}
	var lobby mockLobby
	templateFS := fstest.MapFS{ // tests parseTemplate
		"any-file": new(fstest.MapFile),
	}
	var staticFS fstest.MapFS
	newServerTests := []struct {
		Parameters
		Config
		wantOk bool
		want   *Server
	}{
		{}, // no log
		{ // no tokenizer
			Parameters: Parameters{
				Logger: testLog,
			},
		},
		{ // no userDao
			Parameters: Parameters{
				Logger:    testLog,
				Tokenizer: tokenizer,
			},
		},
		{ // no lobby
			Parameters: Parameters{
				Logger:    testLog,
				Tokenizer: tokenizer,
				UserDao:   userDao,
			},
		},
		{ // no challenge
			Parameters: Parameters{
				Logger:    testLog,
				Tokenizer: tokenizer,
				UserDao:   userDao,
				Lobby:     lobby,
			},
		},
		{ // no staticFS
			Parameters: Parameters{
				Logger:    testLog,
				Tokenizer: tokenizer,
				UserDao:   userDao,
				Lobby:     lobby,
			},
		},
		{ // no templateFS
			Parameters: Parameters{
				Logger:    testLog,
				Tokenizer: tokenizer,
				UserDao:   userDao,
				Lobby:     lobby,
				StaticFS:  staticFS,
			},
		},
		{ // no stopDur
			Parameters: Parameters{
				Logger:     testLog,
				Tokenizer:  tokenizer,
				UserDao:    userDao,
				Lobby:      lobby,
				StaticFS:   staticFS,
				TemplateFS: templateFS,
			},
		},
		{ // bad cacheSec
			Parameters: Parameters{
				Logger:     testLog,
				Tokenizer:  tokenizer,
				UserDao:    userDao,
				Lobby:      lobby,
				StaticFS:   staticFS,
				TemplateFS: templateFS,
			},
			Config: Config{
				StopDur:  1 * time.Hour,
				CacheSec: -1,
			},
		},
		{ // missing httpsPort
			Parameters: Parameters{
				Logger:     testLog,
				Tokenizer:  tokenizer,
				UserDao:    userDao,
				Lobby:      lobby,
				StaticFS:   staticFS,
				TemplateFS: templateFS,
			},
			Config: Config{
				StopDur: 1 * time.Hour,
			},
		},
		{ // missing version
			Parameters: Parameters{
				Logger:     testLog,
				Tokenizer:  tokenizer,
				UserDao:    userDao,
				Lobby:      lobby,
				StaticFS:   staticFS,
				TemplateFS: templateFS,
			},
			Config: Config{
				StopDur:   1 * time.Hour,
				HTTPSPort: 443,
			},
		},
		{ // bad version
			Parameters: Parameters{
				Logger:     testLog,
				Tokenizer:  tokenizer,
				UserDao:    userDao,
				Lobby:      lobby,
				StaticFS:   staticFS,
				TemplateFS: templateFS,
			},
			Config: Config{
				StopDur:   1 * time.Hour,
				HTTPSPort: 443,
				Version:   "almost correct :)",
			},
		},
		{ // bad templateFS
			Parameters: Parameters{
				Logger:     testLog,
				Tokenizer:  tokenizer,
				UserDao:    userDao,
				Lobby:      lobby,
				StaticFS:   staticFS,
				TemplateFS: make(fstest.MapFS),
			},
			Config: Config{
				StopDur:   1 * time.Hour,
				HTTPSPort: 443,
				Version:   "ok",
			},
		},
		{ // happy path
			Parameters: Parameters{
				Logger:     testLog,
				Tokenizer:  tokenizer,
				UserDao:    userDao,
				Lobby:      lobby,
				StaticFS:   staticFS,
				TemplateFS: templateFS,
			},
			Config: Config{
				StopDur:   1 * time.Hour,
				CacheSec:  86400,
				HTTPSPort: 443,
				Version:   "9d2ffad8e5e5383569d37ec381147f2d",
				Challenge: Challenge{
					Token: "a",
					Key:   "b",
				},
				ColorConfig: ColorConfig{
					CanvasPrimary: "blue",
				},
			},
			wantOk: true,
			want: &Server{
				log:   testLog,
				lobby: lobby,
				Config: Config{
					StopDur:   1 * time.Hour,
					CacheSec:  86400,
					HTTPSPort: 443,
					Version:   "9d2ffad8e5e5383569d37ec381147f2d",
					Challenge: Challenge{
						Token: "a",
						Key:   "b",
					},
					ColorConfig: ColorConfig{
						CanvasPrimary: "blue",
					},
				},
			},
		},
	}
	for i, test := range newServerTests {
		got, err := test.Config.NewServer(test.Parameters)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		// cannot use DeepEqual on Server because http.Server and template.Template contain a error fields:
		case !reflect.DeepEqual(test.want.log, got.log),
			!reflect.DeepEqual(test.want.lobby, got.lobby),
			test.want.Challenge != got.Challenge,
			test.want.Config != got.Config:
			t.Errorf("Test %v: server not copied from from arguments properly: %v", i, got)
		default:
			nilChecks := []interface{}{
				got.log,
				got.lobby,
				got.HTTPServer,
				got.HTTPSServer,
			}
			for j, gotJ := range nilChecks {
				if gotJ == nil {
					t.Errorf("Test %v: server left reference %v nil: %v", i, j, gotJ)
				}
			}
		}
	}
}

func TestFileHandler(t *testing.T) {
	const (
		cacheMaxAge = "max-age=???"
		textHTML    = "text/html; charset=utf-8"
		// header constants are duplicated here
		headerCacheControl            = "Cache-Control"
		headerAcceptEncoding          = "Accept-Encoding"
		headerStrictTransportSecurity = "Strict-Transport-Security"
		headerContentEncoding         = "Content-Encoding"
		headerContentType             = "Content-Type"
	)
	handleFileHeadersTests := []struct {
		path          string
		wantHeader    http.Header
		requestHeader http.Header
	}{
		{
			path: "/" + indexHTML,
			wantHeader: http.Header{
				headerCacheControl:            {"no-store"},
				headerStrictTransportSecurity: {cacheMaxAge},
				headerContentType:             {textHTML},
			},
		},
		{
			path: "/" + indexHTML,
			requestHeader: http.Header{
				headerAcceptEncoding: {"gzip"},
				headerContentType:    {textHTML},
			},
			wantHeader: http.Header{
				headerCacheControl:            {"no-store"},
				headerStrictTransportSecurity: {cacheMaxAge},
				headerContentEncoding:         {"gzip"},
				headerContentType:             {textHTML},
			},
		},
		{
			path: "/file.html",
			wantHeader: http.Header{
				headerCacheControl:            {cacheMaxAge},
				headerStrictTransportSecurity: {cacheMaxAge},
				headerContentType:             {textHTML},
			},
		},
	}
	for i, test := range handleFileHeadersTests {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("", test.path, nil)
		r.Header = test.requestHeader
		handlerCalled := false
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})
		fh := fileHandler(h, cacheMaxAge)
		fh.ServeHTTP(w, r)
		gotHeader := w.Header()
		switch {
		case !handlerCalled:
			t.Errorf("Test %v: wanted handler to be called", i)
		case !reflect.DeepEqual(test.wantHeader, gotHeader):
			t.Errorf("Test %v headers not equal\nwanted: %v\ngot:    %v", i, test.wantHeader, gotHeader)
		}
	}
}

func TestHTTPHandler(t *testing.T) {
	httpHandlerTests := []struct {
		Challenge
		httpURI              string
		httpsPort            int
		httpsRedirectHandler http.HandlerFunc
		wantCode             int
		wantBody             string
	}{
		{
			Challenge: Challenge{
				Token: "abc",
				Key:   "def",
			},
			httpURI:  acmeHeader + "abc",
			wantCode: 200,
			wantBody: "abc.def",
		},
		{
			Challenge: Challenge{
				Token: "fred",
			},
			httpURI:  acmeHeader + "barney",
			wantCode: 404,
			wantBody: "404 page not found\n", // flaky check, but ensures actual token.key is not written to body
		},
		{
			httpURI:   "http://example.com/",
			httpsPort: 443,
			httpsRedirectHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(418)
			},
			wantCode: 418,
		},
	}
	for i, test := range httpHandlerTests {
		cfg := Config{
			Challenge: test.Challenge,
			HTTPSPort: test.httpsPort,
		}
		r := httptest.NewRequest("", test.httpURI, nil)
		w := httptest.NewRecorder()
		h := cfg.httpHandler(test.httpsRedirectHandler)
		h.ServeHTTP(w, r)
		gotCode := w.Code
		if test.wantCode != gotCode {
			t.Errorf("Test %v: wanted status code %v, got %v", i, test.wantCode, gotCode)
		}
	}
}

func TestHTTPSHandler(t *testing.T) {
	username := "selene" // used to check token for POST
	monitor := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		t.Errorf("monitory called")
	})
	withTLS := func(r *http.Request) *http.Request {
		r.TLS = new(tls.ConnectionState)
		return r
	}
	withSecHeader := func(r *http.Request) *http.Request {
		r.Header.Add("Sec-Fetch-Mode", "same-origin")
		return r
	}
	withAuthorization := func(r *http.Request) *http.Request {
		r.Header.Add("Authorization", "Bearer GOOD1")
		r.Form = url.Values{"username": {username}}
		return r
	}
	httpsHandlerTests := []struct {
		*http.Request
		Config
		Parameters
		httpHandler          http.HandlerFunc
		httpsRedirectHandler http.HandlerFunc
		*template.Template
		wantCode int
	}{
		{ // acme challenge with no TLS sent to HTTPS
			Request: httptest.NewRequest("GET", acmeHeader, nil),
			httpHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(418)
			},
			wantCode: 418,
		},
		{
			Request: httptest.NewRequest("GET", "/want-redirect", nil),
			httpsRedirectHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(301)
			},
			Config: Config{
				NoTLSRedirect: true,
			},
			wantCode: 301,
		},
		{
			Request: withSecHeader(httptest.NewRequest("GET", "/unknown1", nil)),
			Config: Config{
				NoTLSRedirect: true,
			},
			wantCode: 404,
		},
		{
			Request:  withTLS(httptest.NewRequest("GET", "/unknown2", nil)),
			wantCode: 404,
		},
		{
			Request:  withTLS(httptest.NewRequest("GET", "/", nil)),
			Template: template.Must(template.New(indexHTML).Parse("")),
			wantCode: 200,
		},
		{
			Request: withTLS(withAuthorization(httptest.NewRequest("POST", "/unknown3", nil))),
			Parameters: Parameters{
				Tokenizer: mockTokenizer{
					ReadUsernameFunc: func(tokenString string) (string, error) {
						return username, nil
					},
				},
			},
			wantCode: 404,
		},
		{
			Request:  withTLS(httptest.NewRequest("DELETE", "/", nil)),
			wantCode: 405,
		},
	}
	for i, test := range httpsHandlerTests {
		w := httptest.NewRecorder()
		test.Parameters.UserDao = mockUserDao{
			backendFunc: func() user.Backend {
				return nil
			},
		}
		h := test.Config.httpsHandler(test.httpHandler, test.httpsRedirectHandler, test.Parameters, test.Template, monitor)
		h.ServeHTTP(w, test.Request)
		gotCode := w.Code
		if test.wantCode != gotCode {
			t.Errorf("Test %v: status codes not equal: wanted: %v, got %v", i, test.wantCode, gotCode)
		}
	}
}

func TestCheckTokenUsername(t *testing.T) {
	want := "selene"
	checkTokenUsernameTests := []struct {
		authorizationHeader  string
		tokenUsername        string
		readTokenUsernameErr error
		formUsername         string
		wantOk               bool
	}{
		{},
		{
			authorizationHeader: "bad bearer token",
		},
		{
			authorizationHeader: "Bearer EVIL",
		},
		{
			authorizationHeader:  "Bearer GOOD2",
			readTokenUsernameErr: fmt.Errorf("tokenizer error"),
		},
		{
			authorizationHeader: "Bearer GOOD3",
			formUsername:        "alice",
		},
		{
			authorizationHeader: "Bearer GOOD4",
			formUsername:        want,
			wantOk:              true,
		},
	}
	for i, test := range checkTokenUsernameTests {
		tokenizer := mockTokenizer{
			ReadUsernameFunc: func(tokenString string) (string, error) {
				if test.readTokenUsernameErr != nil {
					return "", test.readTokenUsernameErr
				}
				return want, nil
			},
		}
		r := httptest.NewRequest("", "/", nil)
		r.Header.Add("Authorization", test.authorizationHeader)
		r.Form = make(url.Values)
		r.Form.Add("username", test.formUsername)
		err := checkTokenUsername(r, tokenizer)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		}
	}
}

func TestHTTPError(t *testing.T) {
	w := httptest.NewRecorder()
	want := 400
	httpError(w, want)
	got := w.Code
	switch {
	case want != got:
		t.Errorf("wanted error message to contain %v, got %v", want, got)
	case w.Body.Len() <= 1: // ends in \n character
		t.Errorf("wanted status code info for error (%v) in body", want)
	}
}

func TestWriteInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	err := fmt.Errorf("mock error")
	log := new(logtest.Logger)
	want := 500
	writeInternalError(err, log, w)
	got := w.Code
	switch {
	case want != got:
		t.Errorf("wanted error message to contain %v, got %v", want, got)
	case !strings.Contains(w.Body.String(), err.Error()):
		t.Errorf("wanted message in body (%v), but got %v", err.Error(), w.Body.String())
	case !strings.Contains(log.String(), err.Error()):
		t.Errorf("wanted message in log (%v), but got %v", err.Error(), log.String())
	}
}

func TestHasSecHeader(t *testing.T) {
	hasSecHeaderTests := map[string]bool{
		"Accept":          false,
		"DNT":             false,
		"":                false,
		"inSec-t":         false,
		"Sec-Fetch-Mode:": true,
	}
	for header, want := range hasSecHeaderTests {
		r := httptest.NewRequest("", "/", nil)
		r.Header.Add(header, "")
		got := hasSecHeader(r)
		if want != got {
			t.Errorf("wanted hasSecHeader = %v when header = %v", want, header)
		}
	}
}

func TestAddMimeType(t *testing.T) {
	// the commented-out lines have MIME types that vary by system
	const textHTML = "text/html; charset=utf-8"
	addMimeTypeTests := map[string]string{
		// "LICENSE":       "text/plain; charset=utf-8",
		// "favicon.ico":   "image/vnd.microsoft.icon",
		"favicon.png":         "image/png",
		"favicon.svg":         "image/svg+xml",
		"manifest.json":       "application/json",
		"selene-bananas.wasm": "application/wasm",
		"init.js":             "text/javascript; charset=utf-8",
		"any.html":            textHTML,
		"/" + indexHTML:       textHTML,
	}
	for fileName, want := range addMimeTypeTests {
		w := httptest.NewRecorder()
		addMimeType(fileName, w)
		got := w.Header().Get("Content-Type")
		if want != got {
			t.Errorf("when filename = %v, wanted mimeType %v, got %v", fileName, want, got)
		}
	}
}

func TestTemplateHandler(t *testing.T) {
	serveTemplateTests := []struct {
		templateName string
		templateText string
		path         string
		data         interface{}
		wantCode     int
		wantBody     string
	}{
		{
			templateName: indexHTML,
			path:         "/" + indexHTML,
			templateText: "stuff",
			wantCode:     200,
			wantBody:     "stuff",
		},
		{ // different content type
			templateName: "init.js",
			path:         "/init.js",
			wantCode:     200,
		},
		{
			templateName: "name1.html",
			templateText: "template for {{ . }}",
			path:         "/name1.html",
			data:         "selene",
			wantBody:     "template for selene",
			wantCode:     200,
		},
		{
			templateName: "name2.html",
			templateText: "template for {{ .Name }}",
			path:         "/name2.html",
			data:         struct{ Name string }{Name: "selene"},
			wantBody:     "template for selene",
			wantCode:     200,
		},
		{
			templateName: "name3.html",
			templateText: "template for {{ .NameINVALID }}",
			path:         "/name3.html",
			data:         struct{ Name string }{Name: "selene"},
			wantCode:     500,
		},
	}
	for i, test := range serveTemplateTests {
		template := template.Must(template.New(test.templateName).Parse(test.templateText))
		r := httptest.NewRequest("", test.path, nil)
		w := httptest.NewRecorder()
		log := logtest.DiscardLogger
		h := templateHandler(template, test.data, log)
		h.ServeHTTP(w, r)
		gotCode := w.Code
		gotBody := w.Body.String()
		switch {
		case test.wantCode != gotCode:
			t.Errorf("Test %v: status codes not equal: wanted: %v, got:    %v", i, test.wantCode, gotCode)
		case test.wantBody != gotBody:
			if test.wantCode == 200 {
				t.Errorf("Test %v: body not equal:\nwanted: %v\ngot:    %v", i, test.wantBody, gotBody)
			}
		}
	}
}

func TestHTTPSRedirectHandler(t *testing.T) {
	httpsRedirectHandlerTests := []struct {
		httpURI      string
		httpsPort    int
		wantCode     int
		wantLocation string
	}{
		{
			httpURI:      "http://example1.com/",
			httpsPort:    443,
			wantCode:     301,
			wantLocation: "https://example1.com/",
		},
		{
			httpURI:      "https://example2.com/",
			httpsPort:    443,
			wantCode:     301,
			wantLocation: "https://example2.com/",
		},
		{
			httpURI:      "http://example3.com:80/abc",
			httpsPort:    443,
			wantCode:     301,
			wantLocation: "https://example3.com/abc",
		},
		{
			httpURI:      "http://example4.com:8001/abc/d",
			httpsPort:    8000,
			wantCode:     301,
			wantLocation: "https://example4.com:8000/abc/d",
		},
	}
	for i, test := range httpsRedirectHandlerTests {
		r := httptest.NewRequest("", test.httpURI, nil)
		w := httptest.NewRecorder()
		h := httpsRedirectHandler(test.httpsPort)
		h.ServeHTTP(w, r)
		gotCode := w.Code
		switch {
		case test.wantCode != gotCode:
			t.Errorf("Test %v: wanted status code %v, got %v", i, test.wantCode, gotCode)
		case test.wantLocation != w.Header().Get("Location"):
			t.Errorf("Test %v: Locations no equal:\nwanted: %v\ngot:    %v", i, test.wantLocation, w.Header().Get("Location"))
		}
	}
}

func TestValidHTTPAddr(t *testing.T) {
	validHTTPAddrTests := []struct {
		HTTPPort int
		want     bool
	}{
		{},
		{
			HTTPPort: 80,
			want:     true,
		},
		{
			HTTPPort: 8001,
			want:     true,
		},
	}
	for i, test := range validHTTPAddrTests {
		cfg := Config{
			HTTPPort: test.HTTPPort,
		}
		s := Server{
			Config: cfg,
		}
		got := s.validHTTPAddr()
		if test.want != got {
			t.Errorf("Test %v: when HTTP Port is '%v', validHTTPAddrs not equal: wanted: %v, got: %v", i, test.HTTPPort, test.want, got)
		}
	}
}

func TestWrappedResponseWriter(t *testing.T) {
	w := httptest.NewRecorder()
	var buf bytes.Buffer
	w2 := wrappedResponseWriter{
		Writer:         &buf,
		ResponseWriter: w,
	}
	want := "sent to bb"
	w2.Write([]byte(want))
	got := buf.String()
	if want != got {
		t.Errorf("not equal:\nwanted: %v\ngot:    %v", want, got)
	}
}

func TestGetHandler(t *testing.T) {
	monitor := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// NOOP
	})
	ud := mockUserDao{
		backendFunc: func() user.Backend {
			return nil
		},
	}
	checkCode := func(t *testing.T, path string, p Parameters, cfg Config, template *template.Template, wantCode int) {
		t.Helper()
		r := httptest.NewRequest("", path, nil)
		w := httptest.NewRecorder()
		h := p.getHandler(cfg, template, monitor)
		h.ServeHTTP(w, r)
		if gotCode := w.Code; wantCode != gotCode {
			t.Errorf("codes not equal for GET to %v: status codes not equal: wanted: %v, got: %v", path, wantCode, gotCode)
		}
	}
	t.Run("invalidGetPaths", func(t *testing.T) {
		invalidPaths := []string{"/invalid/get/path", "/ping"}
		for _, path := range invalidPaths {
			var cfg Config
			p := Parameters{
				UserDao: ud,
			}
			checkCode(t, path, p, cfg, nil, 404)
		}
	})
	t.Run("templates", func(t *testing.T) {
		templates := []string{
			"/" + indexHTML,
			"/manifest.json",
			"/serviceWorker.js",
			"/favicon.svg",
			"/network_check.html",
		}
		for _, path := range templates {
			fileName := path[1:]
			template := template.Must(template.New(fileName).Parse(""))
			p := Parameters{
				UserDao: ud,
			}
			var cfg Config
			checkCode(t, path, p, cfg, template, 200)
		}
	})
	t.Run("staticFiles", func(t *testing.T) {
		staticFiles := []string{
			"/wasm_exec.js",
			"/selene-bananas.wasm",
			"/robots.txt",
			"/favicon.png",
			"/favicon.ico",
			"/LICENSE",
		}
		for _, path := range staticFiles {
			fileName := path[1:]
			var cfg Config
			p := Parameters{
				StaticFS: fstest.MapFS{
					fileName: new(fstest.MapFile),
				},
				UserDao: ud,
			}
			checkCode(t, path, p, cfg, nil, 200)
		}
	})
	t.Run("lobby", func(t *testing.T) {
		var cfg Config
		p := Parameters{
			Tokenizer: mockTokenizer{
				ReadUsernameFunc: func(tokenString string) (string, error) {
					return "", nil
				},
			},
			Lobby: mockLobby{
				addUserFunc: func(username string, w http.ResponseWriter, r *http.Request) error {
					return nil
				},
			},
			UserDao: ud,
		}
		checkCode(t, "/lobby", p, cfg, nil, 200)
	})
	t.Run("monitor", func(t *testing.T) {
		var cfg Config
		p := Parameters{
			UserDao: ud,
		}
		// empty monitor used in checkCode
		checkCode(t, "/monitor", p, cfg, nil, 200)
	})
	t.Run("rootHandler", func(t *testing.T) {
		template := template.Must(template.New(indexHTML).Parse(""))
		p := Parameters{
			UserDao: ud,
		}
		var cfg Config
		checkCode(t, "/", p, cfg, template, 200)
	})
}

func TestPostHandler(t *testing.T) {
	type handlePostTest struct {
		path          string
		authorization string
		wantCode      int
	}
	var handlePostTests []handlePostTest
	for _, path := range []string{"/", "/invalid/post/path"} {
		handlePostTests = append(handlePostTests,
			handlePostTest{path: path, wantCode: 403},
			handlePostTest{path: path, wantCode: 404, authorization: "Bearer GOOD5"},
		)
	}
	for _, path := range []string{"/user_create", "/user_login"} {
		handlePostTests = append(handlePostTests,
			handlePostTest{path: path, wantCode: 200},
		)
	}
	for _, path := range []string{"/user_update_password", "/user_delete", "/ping"} {
		handlePostTests = append(handlePostTests,
			handlePostTest{path: path, wantCode: 403},
			handlePostTest{path: path, wantCode: 200, authorization: "Bearer GOOD6"},
		)
	}
	u := "selene"
	formParams := url.Values{
		"username":         {u},
		"password":         {"s3cr3t_old"},
		"password_confirm": {"s3cr3t_new"},
	}
	tokenizer := mockTokenizer{
		CreateFunc: func(username string, points int) (string, error) {
			return "", nil
		},
		ReadUsernameFunc: func(tokenString string) (string, error) {
			return u, nil
		},
	}
	lobby := mockLobby{
		removeUserFunc: func(username string) {
			// NOOP
		},
	}
	userDao := mockUserDao{
		createFunc: func(ctx context.Context, u user.User) error {
			return nil
		},
		loginFunc: func(ctx context.Context, u user.User) (*user.User, error) {
			return new(user.User), nil
		},
		updatePasswordFunc: func(ctx context.Context, u user.User, newP string) error {
			return nil
		},
		deleteFunc: func(ctx context.Context, u user.User) error {
			return nil
		},
	}
	for i, test := range handlePostTests {
		r := httptest.NewRequest("", test.path, nil)
		r.Form = formParams
		r.Header.Add("Authorization", test.authorization)
		w := httptest.NewRecorder()
		p := Parameters{
			Logger:    logtest.DiscardLogger,
			Tokenizer: tokenizer,
			Lobby:     lobby,
			UserDao:   userDao,
		}
		h := p.postHandler()
		h.ServeHTTP(w, r)
		gotCode := w.Code
		if test.wantCode != gotCode {
			t.Errorf("Test %v:\nPOST to %v, authorization='%v': status codes not equal: wanted: %v, got: %v", i, test.path, test.authorization, test.wantCode, gotCode)
		}
	}
}

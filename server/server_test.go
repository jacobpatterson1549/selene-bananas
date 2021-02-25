package server

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/db/user"
)

func TestNewServer(t *testing.T) {
	testLog := log.New(io.Discard, "", 0)
	var tokenizer mockTokenizer
	var userDao mockUserDao
	var lobby mockLobby
	templateFS := fstest.MapFS{ // tests parseTemplate
		"index.html": &fstest.MapFile{Data: []byte{}},
	}
	var staticFS fstest.MapFS
	newServerTests := []struct {
		Config
		*log.Logger
		Tokenizer
		UserDao
		Lobby
		templateFS, staticFS fs.FS
		wantOk               bool
		want                 *Server
	}{
		{}, // no log
		{ // no tokenizer
			Logger: testLog,
		},
		{ // no userDao
			Logger:    testLog,
			Tokenizer: tokenizer,
		},
		{ // no lobby
			Logger:    testLog,
			Tokenizer: tokenizer,
			UserDao:   userDao,
		},
		{ // no templateFS
			Logger:    testLog,
			Tokenizer: tokenizer,
			UserDao:   userDao,
			Lobby:     lobby,
		},
		{ // no staticFS
			Logger:     testLog,
			Tokenizer:  tokenizer,
			UserDao:    userDao,
			Lobby:      lobby,
			templateFS: templateFS,
		},
		{ // no stopDur
			Logger:     testLog,
			Tokenizer:  tokenizer,
			UserDao:    userDao,
			Lobby:      lobby,
			templateFS: templateFS,
			staticFS:   staticFS,
		},
		{ // bad cacheSec
			Logger:     testLog,
			Tokenizer:  tokenizer,
			UserDao:    userDao,
			Lobby:      lobby,
			templateFS: templateFS,
			staticFS:   staticFS,
			Config: Config{
				StopDur:  1 * time.Hour,
				CacheSec: -1,
			},
		},
		{ // missing httpsPort
			Logger:     testLog,
			Tokenizer:  tokenizer,
			UserDao:    userDao,
			Lobby:      lobby,
			templateFS: templateFS,
			staticFS:   staticFS,
			Config: Config{
				StopDur: 1 * time.Hour,
			},
		},
		{ // bad templateFS
			Logger:     testLog,
			Tokenizer:  tokenizer,
			UserDao:    userDao,
			Lobby:      lobby,
			templateFS: make(fstest.MapFS),
			staticFS:   staticFS,
			Config: Config{
				StopDur:   1 * time.Hour,
				HTTPSPort: 443,
			},
		},
		{ // minimum happy path
			Logger:     testLog,
			Tokenizer:  tokenizer,
			UserDao:    userDao,
			Lobby:      lobby,
			templateFS: templateFS,
			staticFS:   staticFS,
			Config: Config{
				StopDur:   1 * time.Hour,
				CacheSec:  86400,
				HTTPSPort: 443,
				ColorConfig: ColorConfig{
					CanvasPrimary: "blue",
				},
			},
			wantOk: true,
			want: &Server{
				log:       testLog,
				tokenizer: tokenizer,
				userDao:   userDao,
				lobby:     lobby,
				Config: Config{
					StopDur:   1 * time.Hour,
					CacheSec:  86400,
					HTTPSPort: 443,
					ColorConfig: ColorConfig{
						CanvasPrimary: "blue",
					},
				},
				cacheMaxAge: "max-age=86400",
			},
		},
	}
	for i, test := range newServerTests {
		got, err := test.Config.NewServer(test.Logger, test.Tokenizer, test.UserDao, test.Lobby, test.templateFS, test.staticFS)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		// cannot use DeepEqual on Server because http.Server and template.Template contain a error fields:
		case !reflect.DeepEqual(test.want.log, got.log),
			!reflect.DeepEqual(test.want.tokenizer, got.tokenizer),
			!reflect.DeepEqual(test.want.userDao, got.userDao),
			!reflect.DeepEqual(test.want.lobby, got.lobby),
			test.want.cacheMaxAge != got.cacheMaxAge,
			test.want.Config != got.Config:
			t.Errorf("Test %v: server not copied from from arguments properly: %v", i, got)
		default:
			nilChecks := []interface{}{
				got.log,
				got.httpServer,
				got.httpsServer,
				got.template,
				got.serveStatic,
				got.monitor,
			}
			for j, gotJ := range nilChecks {
				if gotJ == nil {
					t.Errorf("Test %v: server left reference %v nil: %v", i, j, gotJ)
				}
			}
		}
	}
}

func TestHandleFileVersion(t *testing.T) {
	handleFileVersionTests := []struct {
		version      string
		url          string
		wantCode     int
		wantLocation string
	}{
		{
			url:      "http://example.com/",
			wantCode: 200,
		},
		{
			url:      "http://example.com/main.wasm?v=",
			wantCode: 200,
		},
		{
			version:  "abc",
			url:      "http://example.com/main.wasm?v=abc",
			wantCode: 200,
		},
		{
			version:  "abc",
			url:      "http://example.com/favicon.svg",
			wantCode: 200,
		},
		{
			version:      "abc",
			url:          "http://example.com/main.wasm",
			wantCode:     301,
			wantLocation: "http://example.com/main.wasm?v=abc",
		},
		{
			version:      "abc",
			url:          "http://example.com/main.wasm?v=defg",
			wantCode:     301,
			wantLocation: "http://example.com/main.wasm?v=abc",
		},
	}
	h := func(w http.ResponseWriter, r *http.Request) {
		// NOOP - the version is handled before the handler is called
	}
	hf := http.HandlerFunc(h)
	for i, test := range handleFileVersionTests {
		s := Server{
			Config: Config{
				Version: test.version,
			},
		}
		r := httptest.NewRequest("", test.url, nil)
		w := httptest.NewRecorder()
		s.handleFile(w, r, hf)
		gotCode := w.Code
		gotHeader := w.Header()
		delete(gotHeader, "Cache-Control")
		switch {
		case test.wantCode != gotCode:
			t.Errorf("Test %v: wanted %v status code, got %v", i, test.wantCode, gotCode)
		case test.wantCode == 301 && test.wantLocation != w.Header().Get("Location"):
			t.Errorf("Test %v: wanted Location header %v, got %v", i, test.wantLocation, w.Header().Get("Location"))
		}
	}
}

func TestHandleFile(t *testing.T) {
	cacheMaxAge := "max-age=???"
	handleFileTests := []struct {
		path          string
		wantHeader    http.Header
		requestHeader http.Header
	}{
		{
			path: "/",
			wantHeader: http.Header{
				"Cache-Control": {"no-store"},
				"Content-Type":  {"text/html; charset=utf-8"},
			},
		},
		{
			path: "/",
			requestHeader: http.Header{
				"Accept-Encoding": {"gzip"},
				"Content-Type":    {"text/html; charset=utf-8"},
			},
			wantHeader: http.Header{
				"Cache-Control":    {"no-store"},
				"Content-Encoding": {"gzip"},
				"Content-Type":     {"text/html; charset=utf-8"},
			},
		},
		{
			path: "/file.html",
			wantHeader: http.Header{
				"Cache-Control": {cacheMaxAge},
				"Content-Type":  {"text/html; charset=utf-8"},
			},
		},
	}
	for i, test := range handleFileTests {
		s := Server{
			cacheMaxAge: cacheMaxAge,
		}
		w := httptest.NewRecorder()
		r := httptest.NewRequest("", test.path, nil)
		r.Header = test.requestHeader
		handlerCalled := false
		h := func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		}
		hf := http.HandlerFunc(h)
		s.handleFile(w, r, hf)
		gotHeader := w.Header()
		switch {
		case !handlerCalled:
			t.Errorf("Test %v: wanted handler to be called", i)
		case !reflect.DeepEqual(test.wantHeader, gotHeader):
			t.Errorf("Test %v headers not equal\nwanted: %v\ngot:    %v", i, test.wantHeader, gotHeader)
		}
	}
}

func TestHandleHTTP(t *testing.T) {
	handleHTTPTests := []struct {
		httpURI   string
		httpsAddr string
		want      string
	}{
		{
			httpURI:   "http://example.com",
			httpsAddr: ":443",
			want:      "https://example.com",
		},
		{
			httpURI:   "https://example.com",
			httpsAddr: ":443",
			want:      "https://example.com",
		},
		{
			httpURI:   "http://example.com:80/abc",
			httpsAddr: ":443",
			want:      "https://example.com/abc",
		},
		{
			httpURI:   "http://example.com:8001/abc/d",
			httpsAddr: ":8000",
			want:      "https://example.com:8000/abc/d",
		},
	}
	for i, test := range handleHTTPTests {
		s := Server{
			httpsServer: &http.Server{
				Addr: test.httpsAddr,
			},
		}
		r := httptest.NewRequest("", test.httpURI, nil)
		w := httptest.NewRecorder()
		s.handleHTTP(w, r)
		got := w.Header().Get("Location")
		if test.want != got {
			t.Errorf("test %v:\nwanted: %v\ngot:    %v", i, test.want, got)
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
			authorizationHeader:  "Bearer GOOD",
			readTokenUsernameErr: fmt.Errorf("tokenizer error"),
		},
		{
			authorizationHeader: "Bearer GOOD",
			formUsername:        "alice",
		},
		{
			authorizationHeader: "Bearer GOOD",
			formUsername:        want,
			wantOk:              true,
		},
	}
	for i, test := range checkTokenUsernameTests {
		s := Server{
			tokenizer: mockTokenizer{
				ReadUsernameFunc: func(tokenString string) (string, error) {
					if test.readTokenUsernameErr != nil {
						return "", test.readTokenUsernameErr
					}
					return want, nil
				},
			},
		}
		r := http.Request{
			Header: http.Header{
				"Authorization": {test.authorizationHeader},
			},
			Form: url.Values{
				"username": {test.formUsername},
			},
		}
		err := s.checkTokenUsername(&r)
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: wanted error", i)
		}
	}
}

func TestHTTPError(t *testing.T) {
	w := httptest.NewRecorder()
	var s Server
	want := 400
	s.httpError(w, want)
	got := w.Code
	switch {
	case want != got:
		t.Errorf("wanted error message to contain %v, got %v", want, got)
	case w.Body.Len() <= 1: // ends in \n character
		t.Errorf("wanted status code info for error (%v) in body", want)
	}
}

func TestHandleError(t *testing.T) {
	var buf bytes.Buffer
	w := httptest.NewRecorder()
	err := fmt.Errorf("mock error")
	log := log.New(&buf, "", 0)
	s := Server{
		log: log,
	}
	want := 500
	s.handleError(w, err)
	got := w.Code
	switch {
	case want != got:
		t.Errorf("wanted error message to contain %v, got %v", want, got)
	case !strings.Contains(w.Body.String(), err.Error()):
		t.Errorf("wanted message in body (%v), but got %v", err.Error(), w.Body.String())
	case !strings.Contains(buf.String(), err.Error()):
		t.Errorf("wanted message in log (%v), but got %v", err.Error(), buf.String())
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
	var s Server
	for header, want := range hasSecHeaderTests {
		r := http.Request{
			Header: http.Header{
				header: {},
			},
		}
		got := s.hasSecHeader(&r)
		if want != got {
			t.Errorf("wanted hasSecHeader = %v when header = %v", want, header)
		}
	}
}

func TestAddMimeType(t *testing.T) {
	addMimeTypeTests := map[string]string{
		"favicon.png":   "image/png",
		"favicon.svg":   "image/svg+xml",
		"manifest.json": "application/json",
		"init.js":       "application/javascript",
		"main.wasm":     "application/wasm",
		"LICENSE":       "text/plain; charset=utf-8",
		"any.html":      "text/html; charset=utf-8",
		"/":             "text/html; charset=utf-8",
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

func TestServeTemplate(t *testing.T) {
	serveTemplateTests := []struct {
		templateName    string
		templateText    string
		path            string
		data            interface{}
		wantStatusCode  int
		wantContentType string
		wantBody        string
	}{
		{
			path:            "/unknown",
			wantStatusCode:  500,
			wantContentType: "text/plain; charset=utf-8",
		},
		{ // empty path should go to index.html
			path:            "/",
			templateName:    "index.html",
			templateText:    "stuff",
			wantStatusCode:  200,
			wantContentType: "text/html; charset=utf-8",
			wantBody:        "stuff",
		},
		{ // different content type
			templateName:    "init.js",
			path:            "/init.js",
			wantStatusCode:  200,
			wantContentType: "application/javascript",
		},
		{
			templateName:    "name.html",
			templateText:    "template for {{ . }}",
			path:            "/name.html",
			data:            "selene",
			wantStatusCode:  200,
			wantContentType: "text/html; charset=utf-8",
			wantBody:        "template for selene",
		},
	}
	for i, test := range serveTemplateTests {
		s := Server{
			log:      log.New(io.Discard, "", 0),
			template: template.Must(template.New(test.templateName).Parse(test.templateText)),
			data:     test.data,
		}
		w := httptest.NewRecorder()
		r := http.Request{
			URL: &url.URL{
				Path: test.path,
			},
		}
		h := s.serveTemplate(test.path[1:])
		h.ServeHTTP(w, &r)
		switch {
		case test.wantStatusCode != w.Code:
			t.Errorf("Test %v: status codes not equal: wanted: %v, got:    %v", i, test.wantStatusCode, w.Code)
		case test.wantContentType != w.Header().Get("Content-Type"):
			t.Errorf("Test %v: headers content types not equal:\nwanted: %v\ngot:    %v", i, test.wantContentType, w.Header().Get("Content-Type"))
		case test.wantStatusCode == 200 && test.wantBody != w.Body.String():
			t.Errorf("Test %v: body not equal:\nwanted: %v\ngot:    %v", i, test.wantBody, w.Body.String())
		}
	}
}

func TestValidHTTPAddr(t *testing.T) {
	validHTTPAddrTests := []struct {
		addr string
		want bool
	}{
		{},
		{
			addr: "example.com",
			want: true,
		},
		{
			addr: ":8001",
			want: true,
		},
	}
	for i, test := range validHTTPAddrTests {
		s := Server{
			httpServer: &http.Server{
				Addr: test.addr,
			},
		}
		got := s.validHTTPAddr()
		if test.want != got {
			t.Errorf("Test %v: wanted %v, got %v for when addr is '%v'", i, test.want, got, test.addr)
		}
	}
}

func TestWrappedResponseWriter(t *testing.T) {
	w := httptest.NewRecorder()
	var bb bytes.Buffer
	w2 := wrappedResponseWriter{
		Writer:         &bb,
		ResponseWriter: w,
	}
	want := "sent to bb"
	w2.Write([]byte(want))
	got := bb.String()
	if want != got {
		t.Errorf("not equal:\nwanted: %v\ngot:    %v", want, got)
	}
}

func TestHandleGet(t *testing.T) {
	type handleGetTest struct {
		path     string
		wantCode int
	}
	var handleGetTests []handleGetTest
	for _, path := range []string{"/invalid/get/path", "/ping"} {
		handleGetTests = append(handleGetTests,
			handleGetTest{path: path, wantCode: 404},
		)
	}
	validGetEndpoints := []string{
		"/",
		"/manifest.json",
		"/serviceWorker.js",
		"/favicon.svg",
		"/network_check.html",
		"/wasm_exec.js",
		"/main.wasm",
		"/robots.txt",
		"/favicon.png",
		"/LICENSE",
		"/lobby",
		"/monitor",
	}
	for _, path := range validGetEndpoints {
		handleGetTests = append(handleGetTests,
			handleGetTest{path: path, wantCode: 200},
		)
	}
	for i, test := range handleGetTests {
		r := httptest.NewRequest("", test.path+"?v=1", nil)
		w := httptest.NewRecorder()
		tmplName := test.path[1:]
		if len(tmplName) == 0 {
			tmplName = "index.html"
		}
		tmpl := template.Must(template.New(tmplName).Parse(""))
		s := Server{
			log: log.New(io.Discard, "", 0),
			tokenizer: mockTokenizer{
				ReadUsernameFunc: func(tokenString string) (string, error) {
					return "", nil
				},
			},
			lobby: mockLobby{
				addUserFunc: func(username string, w http.ResponseWriter, r *http.Request) error {
					return nil
				},
			},
			template: tmpl,
			serveStatic: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				// NOOP
			}),
			monitor: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				// NOOP
			}),
			Config: Config{
				Version: "1",
			},
		}
		s.handleGet(w, r)
		gotCode := w.Code
		if test.wantCode != gotCode {
			t.Errorf("Test %v:\nGET to %v: status codes not equal: wanted: %v, got: %v", i, test.path, test.wantCode, gotCode)
		}
	}
}

func TestHandlePost(t *testing.T) {
	type handlePostTest struct {
		path          string
		authorization string
		wantCode      int
	}
	var handlePostTests []handlePostTest
	for _, path := range []string{"/", "/invalid/post/path"} {
		handlePostTests = append(handlePostTests,
			handlePostTest{path: path, wantCode: 403},
			handlePostTest{path: path, wantCode: 404, authorization: "Bearer GOOD"},
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
			handlePostTest{path: path, wantCode: 200, authorization: "Bearer GOOD"},
		)
	}
	u := "selene"
	formParams := url.Values{
		"username":         {u},
		"password":         {"s3cr3t_old"},
		"password_confirm": {"s3cr3t_new"},
	}
	mockTokenizer := mockTokenizer{
		CreateFunc: func(username string, points int) (string, error) {
			return "", nil
		},
		ReadUsernameFunc: func(tokenString string) (string, error) {
			return u, nil
		},
	}
	mockLobby := mockLobby{
		removeUserFunc: func(username string) {
			// NOOP
		},
	}
	mockUserDao := mockUserDao{
		createFunc: func(ctx context.Context, u user.User) error {
			return nil
		},
		readFunc: func(ctx context.Context, u user.User) (*user.User, error) {
			return &user.User{}, nil
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
		s := Server{
			log:       log.New(io.Discard, "", 0),
			tokenizer: mockTokenizer,
			lobby:     mockLobby,
			userDao:   mockUserDao,
		}
		s.handlePost(w, r)
		gotCode := w.Code
		if test.wantCode != gotCode {
			t.Errorf("Test %v:\nPOST to %v, authorization='%v': status codes not equal: wanted: %v, got: %v", i, test.path, test.authorization, test.wantCode, gotCode)
		}
	}
}

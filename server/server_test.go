package server

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

type mockTokenizer struct {
	CreateFunc       func(username string, points int) (string, error)
	ReadUsernameFunc func(tokenString string) (string, error)
}

func (t mockTokenizer) Create(username string, points int) (string, error) {
	return t.CreateFunc(username, points)
}

func (t mockTokenizer) ReadUsername(tokenString string) (string, error) {
	return t.ReadUsernameFunc(tokenString)
}

func TestHandleFileVersion(t *testing.T) {
	handleFileVersionTests := []struct {
		version    string
		url        string
		wantCode   int
		wantHeader http.Header
	}{
		{
			url:        "http://example.com/main.wasm?v=",
			wantCode:   200,
			wantHeader: make(http.Header),
		},
		{
			version:    "abc",
			url:        "http://example.com/main.wasm?v=abc",
			wantCode:   200,
			wantHeader: make(http.Header),
		},
		{
			version:    "abc",
			url:        "http://example.com/favicon.svg",
			wantCode:   200,
			wantHeader: make(http.Header),
		},
		{
			version:  "abc",
			url:      "http://example.com/main.wasm",
			wantCode: 301,
			wantHeader: http.Header{
				"Location": {"http://example.com/main.wasm?v=abc"},
			},
		},
		{
			version:  "abc",
			url:      "http://example.com/main.wasm?v=defg",
			wantCode: 301,
			wantHeader: http.Header{
				"Location": {"http://example.com/main.wasm?v=abc"},
			},
		},
	}
	h := func(w http.ResponseWriter, r *http.Request) {}
	for i, test := range handleFileVersionTests {
		s := Server{
			Config: Config{
				Version: test.version,
			},
		}
		r := httptest.NewRequest("", test.url, nil)
		w := httptest.NewRecorder()
		s.handleFile(w, r, h)
		gotCode := w.Code
		gotHeader := w.Header()
		delete(gotHeader, "Cache-Control")
		switch {
		case test.wantCode != gotCode:
			t.Errorf("Test %v: wanted %v status code, got %v", i, test.wantCode, gotCode)
		case !reflect.DeepEqual(test.wantHeader, gotHeader):
			t.Errorf("Test %v: different headers:\nwanted %+v\ngot    %v", i, test.wantHeader, gotHeader)
		}
	}
}

func TestHandleFile(t *testing.T) {
	s := Server{
		cacheMaxAge: "max-age=???",
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("", "/file", nil)
	r.Header = http.Header{
		"Accept-Encoding": {"gzip"},
	}
	handlerCalled := false
	h := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		fmt.Fprint(w, "test gzipped message")
	}
	s.handleFile(w, r, h)
	switch {
	case w.Header().Get("Cache-Control") != s.cacheMaxAge:
		t.Error("missing cache response header")
	case w.Header().Get("Content-Encoding") != "gzip":
		t.Error("missing gzip response header")
	case !handlerCalled:
		t.Error("wanted handler to be called")
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
	s := Server{
		Config: Config{
			Log: log.New(&buf, "", log.LstdFlags),
		},
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
			Header: map[string][]string{
				header: nil,
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
		"/":             "", // no mime type
	}
	for fileName, want := range addMimeTypeTests {
		var s Server
		w := httptest.NewRecorder()
		s.addMimeType(fileName, w)
		got := w.Header().Get("Content-Type")
		if want != got {
			t.Errorf("when filename = %v, wanted mimeType %v, got %v", fileName, want, got)
		}
	}
}

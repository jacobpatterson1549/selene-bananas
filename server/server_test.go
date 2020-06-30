package server

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

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
	okHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	for i, test := range handleFileVersionTests {
		s := Server{
			version: test.version,
		}
		fileHandler := s.handleFile(okHandler, true)
		r := httptest.NewRequest("", test.url, nil)
		w := httptest.NewRecorder()
		fileHandler(w, r)
		gotCode := w.Code
		gotHeader := w.Header()
		switch {
		case test.wantCode != gotCode:
			t.Errorf("Test %v: wanted %v status code, got %v", i, test.wantCode, gotCode)
		case !reflect.DeepEqual(test.wantHeader, gotHeader):
			t.Errorf("Test %v: different headers:\nwanted %+v\ngot    %v", i, test.wantHeader, gotHeader)
		}
	}
}
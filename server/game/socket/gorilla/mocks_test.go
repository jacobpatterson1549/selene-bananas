package gorilla

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type MockHijacker struct {
	http.ResponseWriter
	net.Conn
	*bufio.ReadWriter
}

func (m MockHijacker) Header() http.Header {
	return m.ResponseWriter.Header()
}

func (m MockHijacker) Write(p []byte) (int, error) {
	return m.ReadWriter.Write(p)
}

func (m MockHijacker) WriteHeader(statusCode int) {
	m.ResponseWriter.WriteHeader(statusCode)
}

func (m MockHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return m.Conn, m.ReadWriter, nil
}

type RedirectConn struct {
	net.Conn
	io.Writer
}

func (c RedirectConn) Write(p []byte) (int, error) {
	return c.Writer.Write(p)
}

func newWebsocketResponseWriter() http.ResponseWriter {
	w := httptest.NewRecorder()
	client, _ := net.Pipe()
	sr := strings.NewReader("reader")
	br := bufio.NewReader(sr)
	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	rw := bufio.NewReadWriter(br, bw)
	rc := RedirectConn{
		Conn:   client,
		Writer: bw,
	}
	h := MockHijacker{
		Conn:           rc,
		ReadWriter:     rw,
		ResponseWriter: w,
	}
	return &h
}

func newWebsocketRequest() *http.Request {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Add("Connection", "upgrade")
	r.Header.Add("Upgrade", "websocket")
	r.Header.Add("Sec-Websocket-Version", "13")
	r.Header.Add("Sec-WebSocket-Key", "3D8mi1hwk11RYYWU8rsdIg==")
	return r
}

func newConnWithMocks(t *testing.T) *Conn {
	w := newWebsocketResponseWriter()
	r := newWebsocketRequest()
	u := NewUpgrader()
	conn, err := u.Upgrade(w, r)
	if err != nil {
		t.Fatalf("creating Conn: %v", err)
	}
	return conn
}

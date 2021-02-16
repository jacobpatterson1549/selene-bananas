package socket

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
)

type mockAddr string

func (a mockAddr) Network() string {
	return string(a) + "_NETWORK"
}

func (a mockAddr) String() string {
	return string(a)
}

type mockConn struct {
	ReadMessageFunc      func(m *message.Message) error
	WriteMessageFunc     func(m message.Message) error
	SetReadDeadlineFunc  func(t time.Time) error
	SetWriteDeadlineFunc func(t time.Time) error
	SetPongHandlerFunc   func(h func(appDauta string) error)
	CloseFunc            func() error
	WritePingFunc        func() error
	WriteCloseFunc       func(reason string) error
	IsNormalCloseFunc    func(err error) bool
	RemoteAddrFunc       func() net.Addr
}

func (c *mockConn) ReadMessage(m *message.Message) error {
	return c.ReadMessageFunc(m)
}

func (c *mockConn) WriteMessage(m message.Message) error {
	return c.WriteMessageFunc(m)
}

func (c *mockConn) SetReadDeadline(t time.Time) error {
	return c.SetReadDeadlineFunc(t)
}

func (c *mockConn) SetWriteDeadline(t time.Time) error {
	return c.SetWriteDeadlineFunc(t)
}

func (c *mockConn) SetPongHandler(h func(appData string) error) {
	c.SetPongHandlerFunc(h)
}

func (c *mockConn) Close() error {
	return c.CloseFunc()
}

func (c *mockConn) WritePing() error {
	return c.WritePingFunc()
}

func (c *mockConn) WriteClose(reason string) error {
	return c.WriteCloseFunc(reason)
}

func (c *mockConn) IsNormalClose(err error) bool {
	return c.IsNormalCloseFunc(err)
}

func (c *mockConn) RemoteAddr() net.Addr {
	return c.RemoteAddrFunc()
}

type mockUpgrader func(w http.ResponseWriter, r *http.Request) (Conn, error)

func (u mockUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (Conn, error) {
	return u(w, r)
}

func mockAddUserRequest(playerName string) (player.Name, http.ResponseWriter, *http.Request) {
	pn := player.Name(playerName)
	var w http.ResponseWriter
	var r *http.Request
	return pn, w, r
}

type MockHijacker struct {
	http.ResponseWriter
	net.Conn
	*bufio.ReadWriter
}

func (h MockHijacker) Header() http.Header {
	return h.ResponseWriter.Header()
}

func (h MockHijacker) Write(p []byte) (int, error) {
	return h.ReadWriter.Write(p)
}

func (h MockHijacker) WriteHeader(statusCode int) {
	h.ResponseWriter.WriteHeader(statusCode)
}

func (h MockHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return h.Conn, h.ReadWriter, nil
}

type RedirectConn struct {
	net.Conn
	io.Writer
}

func (w RedirectConn) Write(p []byte) (int, error) {
	return w.Writer.Write(p)
}

func newMockSocketWebSocketResponse() http.ResponseWriter {
	w := httptest.NewRecorder()
	client, _ := net.Pipe()
	sr := strings.NewReader("reader")
	br := bufio.NewReader(sr)
	var bb bytes.Buffer
	bw := bufio.NewWriter(&bb)
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

func newMockWebSocketRequest() *http.Request {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Add("Connection", "upgrade")
	r.Header.Add("Upgrade", "websocket")
	r.Header.Add("Sec-Websocket-Version", "13")
	r.Header.Add("Sec-WebSocket-Key", "3D8mi1hwk11RYYWU8rsdIg==")
	return r
}

func newGorillaConnWithMocks(t *testing.T) *gorillaConn {
	w := newMockSocketWebSocketResponse()
	r := newMockWebSocketRequest()
	u := newGorillaUpgrader()
	conn, err := u.Upgrade(w, r)
	if err != nil {
		t.Fatal("creating gorillaConn")
	}
	return conn.(*gorillaConn)
}

// mockConnReadMessage reads the src message into the destination value using reflection.
func mockConnReadMessage(dest *message.Message, src message.Message) {
	srcV := reflect.ValueOf(src)
	destV := reflect.ValueOf(dest)
	destVE := destV.Elem()
	destVE.Set(srcV)
}

// ReadMinimalMessage reads a message into the json that will not cause an error.
func mockConnReadMinimalMessage(dest *message.Message) {
	src := message.Message{
		Game: &game.Info{},
	}
	mockConnReadMessage(dest, src)
}

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

func (m mockAddr) Network() string {
	return string(m) + "_NETWORK"
}

func (m mockAddr) String() string {
	return string(m)
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

func (m *mockConn) ReadMessage(msg *message.Message) error {
	return m.ReadMessageFunc(msg)
}

func (m *mockConn) WriteMessage(msg message.Message) error {
	return m.WriteMessageFunc(msg)
}

func (m *mockConn) SetReadDeadline(t time.Time) error {
	return m.SetReadDeadlineFunc(t)
}

func (m *mockConn) SetWriteDeadline(t time.Time) error {
	return m.SetWriteDeadlineFunc(t)
}

func (m *mockConn) SetPongHandler(h func(appData string) error) {
	m.SetPongHandlerFunc(h)
}

func (m *mockConn) Close() error {
	return m.CloseFunc()
}

func (m *mockConn) WritePing() error {
	return m.WritePingFunc()
}

func (m *mockConn) WriteClose(reason string) error {
	return m.WriteCloseFunc(reason)
}

func (m *mockConn) IsNormalClose(err error) bool {
	return m.IsNormalCloseFunc(err)
}

func (m *mockConn) RemoteAddr() net.Addr {
	return m.RemoteAddrFunc()
}

type mockUpgrader func(w http.ResponseWriter, r *http.Request) (Conn, error)

func (m mockUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (Conn, error) {
	return m(w, r)
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

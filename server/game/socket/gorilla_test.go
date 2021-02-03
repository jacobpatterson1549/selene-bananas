package socket

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

type (
	MockHijacker struct {
		http.ResponseWriter
		net.Conn
		*bufio.ReadWriter
	}

	RedirectConn struct {
		net.Conn
		io.Writer
	}
)

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

func (w RedirectConn) Write(p []byte) (int, error) {
	return w.Writer.Write(p)
}

func newWebSocketResponse() http.ResponseWriter {
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

func newWebSocketRequest() *http.Request {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Add("Connection", "upgrade")
	r.Header.Add("Upgrade", "websocket")
	r.Header.Add("Sec-Websocket-Version", "13")
	r.Header.Add("Sec-WebSocket-Key", "3D8mi1hwk11RYYWU8rsdIg==")
	return r
}

func newGorillaConn(t *testing.T) *gorillaConn {
	w := newWebSocketResponse()
	r := newWebSocketRequest()
	u := newGorillaUpgrader()
	conn, err := u.Upgrade(w, r)
	if err != nil {
		t.Fatal("creating gorillaConn")
	}
	return conn.(*gorillaConn)
}

func TestGorillaUpgraderUpgrade(t *testing.T) {
	upgradeTests := []struct {
		w       http.ResponseWriter
		r       *http.Request
		wantErr bool
	}{
		{
			w:       &httptest.ResponseRecorder{},
			r:       &http.Request{},
			wantErr: true,
		},
		{
			w: newWebSocketResponse(),
			r: newWebSocketRequest(),
		},
	}
	for i, test := range upgradeTests {
		u := newGorillaUpgrader()
		conn, err := u.Upgrade(test.w, test.r)
		switch {
		case test.wantErr:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		default:
			switch conn.(type) {
			case *gorillaConn:
				if err != nil {
					t.Errorf("Test %v: unwanted error: %v", i, err)
				}
			default:
				t.Errorf("Test %v: wanted Conn to be %T, but was %T", i, gorillaConn{}, conn)
			}
		}
	}
}

func TestGorillaConnIsUnexpectedCloseError(t *testing.T) {
	isUnexpectedCloseErrorTests := []struct {
		err  error
		want bool
	}{
		{},
		{
			err: errors.New("unexpectedCloseError"),
		},
		{
			err: errSocketClosed,
		},
		{
			err: &websocket.CloseError{
				Code: websocket.CloseGoingAway,
				Text: "[desired closure]",
			},
		},
		{
			err: &websocket.CloseError{
				Code: websocket.CloseNoStatusReceived,
				Text: "[normal closure]",
			},
		},
		{
			err: &websocket.CloseError{
				Code: websocket.CloseAbnormalClosure,
				Text: "[an abnormal closure is unexpected]",
			},
			want: true,
		},
	}
	for i, test := range isUnexpectedCloseErrorTests {
		var conn gorillaConn
		got := conn.IsUnexpectedCloseError(test.err)
		if test.want != got {
			t.Errorf("Test %v: wanted IsUnexpectedCloseError to be %v for '%v'", i, test.want, test.err)
		}
	}
}

func TestGorillaConnRemoteAddr(t *testing.T) {
	conn := newGorillaConn(t)
	got := conn.RemoteAddr() // net/pipeAddr
	if got == nil {
		t.Error("wanted non-nil remote address")
	}
}

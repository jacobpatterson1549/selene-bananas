package socket

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
)

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
			w: newMockSocketWebSocketResponse(),
			r: newMockWebSocketRequest(),
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

func TestGorillaConnIsNormalClose(t *testing.T) {
	isNormalCloseTests := []struct {
		err     error
		wantErr bool
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
			wantErr: true,
		},
	}
	for i, test := range isNormalCloseTests {
		var conn gorillaConn
		got := conn.IsNormalClose(test.err)
		if test.wantErr == got {
			t.Errorf("Test %v: wanted isNormalClose to be %v for '%v'", i, test.wantErr, test.err)
		}
	}
}

func TestGorillaConnRemoteAddr(t *testing.T) {
	conn := newGorillaConnWithMocks(t)
	got := conn.RemoteAddr() // net/pipeAddr
	if got == nil {
		t.Error("wanted non-nil remote address")
	}
}

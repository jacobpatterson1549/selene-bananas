package gorilla

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
)

func TestUpgraderUpgrade(t *testing.T) {
	upgradeTests := []struct {
		w      http.ResponseWriter
		r      *http.Request
		wantOk bool
	}{
		{
			w: new(httptest.ResponseRecorder),
			r: httptest.NewRequest("", "/", nil),
		},
		{
			w:      newWebsocketResponseWriter(),
			r:      newWebsocketRequest(),
			wantOk: true,
		},
	}
	for i, test := range upgradeTests {
		u := NewUpgrader()
		conn, err := u.Upgrade(test.w, test.r)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case conn == nil:
			t.Errorf("Test %v: wanted connection", i)
		}
	}
}

func TestConnIsNormalClose(t *testing.T) {
	isNormalCloseTests := []struct {
		err  error
		want bool
	}{
		{},
		{
			err: errors.New("unexpectedCloseError"),
		},
		{
			err: &websocket.CloseError{
				Code: websocket.CloseGoingAway,
			},
			want: true,
		},
		{
			err: &websocket.CloseError{
				Code: websocket.CloseNoStatusReceived,
			},
			want: true,
		},
		{
			err: &websocket.CloseError{
				Code: websocket.CloseAbnormalClosure,
			},
		},
	}
	for i, test := range isNormalCloseTests {
		var conn Conn
		got := conn.IsNormalClose(test.err)
		if test.want != got {
			t.Errorf("Test %v: wanted isNormalClose to be %v for '%v'", i, test.want, test.err)
		}
	}
}

func TestConnRemoteAddr(t *testing.T) {
	conn := newConnWithMocks(t)
	got := conn.RemoteAddr() // net/pipeAddr
	if got == nil {
		t.Error("wanted non-nil remote address")
	}
}

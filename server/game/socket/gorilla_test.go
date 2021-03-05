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
		w      http.ResponseWriter
		r      *http.Request
		wantOk bool
	}{
		{
			w: &httptest.ResponseRecorder{},
			r: httptest.NewRequest("", "/", nil),
		},
		{
			w:      newMockSocketWebSocketResponse(),
			r:      newMockWebSocketRequest(),
			wantOk: true,
		},
	}
	for i, test := range upgradeTests {
		u := newGorillaUpgrader()
		conn, err := u.Upgrade(test.w, test.r)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		default:
			_ = conn.(*gorillaConn)
		}
	}
}

func TestGorillaConnIsNormalClose(t *testing.T) {
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
		var conn gorillaConn
		got := conn.IsNormalClose(test.err)
		if test.want != got {
			t.Errorf("Test %v: wanted isNormalClose to be %v for '%v'", i, test.want, test.err)
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

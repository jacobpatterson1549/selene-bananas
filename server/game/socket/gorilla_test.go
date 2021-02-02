package socket

import (
	"errors"
	"testing"

	"github.com/gorilla/websocket"
)

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

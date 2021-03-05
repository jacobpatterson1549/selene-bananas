package socket

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
)

type (
	// gorillaUpgrader implements the socket.Upgrader interface by wrapping a gorilla/websocket upgrader.
	gorillaUpgrader struct {
		*websocket.Upgrader
	}

	// gorillaConn implements the Conn interface by wrapping a gorilla/websocket GorillaConnection.
	gorillaConn struct {
		*websocket.Conn
	}
)

// NewGorillaUpgrader returns a upgrader tha creates gorilla websocket connections.
func newGorillaUpgrader() *gorillaUpgrader {
	u := new(websocket.Upgrader)
	return &gorillaUpgrader{u}
}

// Upgrade creates a Conn from the http request.
func (u *gorillaUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (Conn, error) {
	c, err := u.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return &gorillaConn{c}, nil
}

// ReadMessage reads the next message from the GorillaConnection.
func (c *gorillaConn) ReadMessage(m *message.Message) error {
	return c.Conn.ReadJSON(m)
}

// WriteJMessage writes the message as json to the GorillaConnection.
func (c *gorillaConn) WriteMessage(m message.Message) error {
	return c.Conn.WriteJSON(m)
}

// WritePing writes a ping message on the GorillaConnection.
func (c *gorillaConn) WritePing() error {
	return c.Conn.WriteMessage(websocket.PingMessage, nil)
}

// WriteClose writes a close message on the connection.  The connestion is NOT closed.
func (c *gorillaConn) WriteClose(reason string) (err error) {
	data := websocket.FormatCloseMessage(websocket.CloseNormalClosure, reason)
	return c.Conn.WriteMessage(websocket.CloseMessage, data)
}

// IsNormalClose determines if the error message is not an unexpected close error.
func (*gorillaConn) IsNormalClose(err error) bool {
	_, ok := err.(*websocket.CloseError) // only errors from gorilla can be normal close errors
	return ok && !websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNoStatusReceived)
}

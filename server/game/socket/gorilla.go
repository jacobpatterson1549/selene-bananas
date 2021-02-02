package socket

import (
	"net"
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

// ReadJSON reads the next json message from the GorillaConnection.
func (c *gorillaConn) ReadJSON(m *message.Message) error {
	return c.ReadJSON(m)
}

// WriteJSON writes the message as json to the GorillaConnection.
func (c *gorillaConn) WriteJSON(m message.Message) error {
	return c.Conn.WriteJSON(m)
}

// Close closes the underlying GorillaConnection.
func (c *gorillaConn) Close() error {
	return c.Conn.Close()
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

// IsUnexpectedCloseError determines if the error message is an unexpected close error.
func (*gorillaConn) IsUnexpectedCloseError(err error) bool {
	return websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNoStatusReceived)
}

// RemoteAddr gets the remote network address of the GorillaConnection.
func (c *gorillaConn) RemoteAddr() net.Addr {
	return c.Conn.RemoteAddr()
}

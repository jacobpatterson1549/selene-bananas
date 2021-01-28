package socket

import (
	"fmt"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
)

type (
	// gorillaUpgrader implements the socket.Upgrader interface by wrapping a gorilla/websocket upgrader.
	gorillaUpgrader struct {
		Upgrader *websocket.Upgrader
	}

	// gorillaConn implements the Conn interface by wrapping a gorilla/websocket GorillaConnection.
	gorillaConn struct {
		Conn *websocket.Conn
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
func (c *gorillaConn) ReadJSON(v interface{}) error {
	return c.ReadJSON(v)
}

// WriteJSON writes the message as json to the GorillaConnection.
func (c *gorillaConn) WriteJSON(v interface{}) error {
	return c.Conn.WriteJSON(v)
}

// Close closes the underlying GorillaConnection.
func (c *gorillaConn) Close() error {
	return c.Conn.Close()
}

// WritePing writes a ping message on the GorillaConnection.
func (c *gorillaConn) WritePing() error {
	return c.Conn.WriteMessage(websocket.PingMessage, nil)
}

// WriteClose writes a close message on the GorillaConnection and always closes it.
func (c *gorillaConn) WriteClose(reason string) (err error) {
	defer func() {
		err2 := c.Close()
		if err == nil {
			err = err2
		}
	}()
	data := websocket.FormatCloseMessage(websocket.CloseNormalClosure, reason)
	if err := c.Conn.WriteMessage(websocket.CloseMessage, data); err != nil {
		return fmt.Errorf("closing GorillaConnection: writing close message: %w", err)
	}
	return nil
}

// IsUnexpectedCloseError determines if the error message is an unexpected close error.
func (*gorillaConn) IsUnexpectedCloseError(err error) bool {
	return websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNoStatusReceived)
}

// RemoteAddr gets the remote network address of the GorillaConnection.
func (c *gorillaConn) RemoteAddr() net.Addr {
	return c.Conn.RemoteAddr()
}

// Package gorilla implements a websocket connection by wrapping gorilla/websocket.
package gorilla

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
)

type (
	// Upgrader implements the socket.Upgrader interface by wrapping a gorilla/websocket Upgrader.
	Upgrader struct {
		*websocket.Upgrader
	}

	// Conn implements the Conn interface by wrapping a gorilla/websocket GorillaConnection.
	Conn struct {
		*websocket.Conn
	}
)

// NewUpgrader returns a upgrader tha creates gorilla websocket connections.
func NewUpgrader() *Upgrader {
	u := new(websocket.Upgrader)
	return &Upgrader{u}
}

// Upgrade creates a Conn from the http request.
func (u *Upgrader) Upgrade(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	c, err := u.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return &Conn{c}, nil
}

// ReadMessage reads the next message from the GorillaConnection.
func (c *Conn) ReadMessage(m *message.Message) error {
	return c.Conn.ReadJSON(m)
}

// WriteMessage writes the message as json to the GorillaConnection.
func (c *Conn) WriteMessage(m message.Message) error {
	return c.Conn.WriteJSON(m)
}

// WritePing writes a ping message on the GorillaConnection.
func (c *Conn) WritePing() error {
	return c.Conn.WriteMessage(websocket.PingMessage, nil)
}

// WriteClose writes a close message on the connection.  The connestion is NOT closed.
func (c *Conn) WriteClose(reason string) (err error) {
	data := websocket.FormatCloseMessage(websocket.CloseNormalClosure, reason)
	return c.Conn.WriteMessage(websocket.CloseMessage, data)
}

// IsNormalClose determines if the error message is not an unexpected close error.
func (*Conn) IsNormalClose(err error) bool {
	_, ok := err.(*websocket.CloseError) // only errors from gorilla can be normal close errors
	return ok && !websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNoStatusReceived)
}

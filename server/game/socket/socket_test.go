package socket

import "net"

type mockConn struct {
	ReadJSONFunc               func(v interface{}) error
	WriteJSONFunc              func(v interface{}) error
	CloseFunc                  func() error
	WritePingFunc              func() error
	WriteCloseFunc             func(reason string) error
	IsUnexpectedCloseErrorFunc func(err error) bool
	RemoteAddrFunc             func() net.Addr
}

func (c *mockConn) ReadJSON(v interface{}) error {
	return c.ReadJSONFunc(v)
}

func (c *mockConn) WriteJSON(v interface{}) error {
	return c.WriteJSONFunc(v)
}

func (c *mockConn) Close() error {
	return c.CloseFunc()
}

func (c *mockConn) WritePing() error {
	return c.WritePingFunc()
}

func (c *mockConn) WriteClose(reason string) error {
	return c.WriteCloseFunc(reason)
}

func (c *mockConn) IsUnexpectedCloseError(err error) bool {
	return c.IsUnexpectedCloseErrorFunc(err)
}

func (c *mockConn) RemoteAddr() net.Addr {
	return c.RemoteAddrFunc()
}

type mockAddr string

func (a mockAddr) Network() string {
	return string(a)
}

func (a mockAddr) String() string {
	return string(a)
}

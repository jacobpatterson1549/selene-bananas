package socket

import (
	"net"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/game/message"
)

// mockAddr implements the net.Addr interface
type mockAddr string

func (m mockAddr) Network() string {
	return string(m) + "_NETWORK"
}

func (m mockAddr) String() string {
	return string(m)
}

type mockConn struct {
	ReadMessageFunc      func(m *message.Message) error
	WriteMessageFunc     func(m message.Message) error
	SetReadDeadlineFunc  func(t time.Time) error
	SetWriteDeadlineFunc func(t time.Time) error
	SetPongHandlerFunc   func(h func(appDauta string) error)
	CloseFunc            func() error
	WritePingFunc        func() error
	WriteCloseFunc       func(reason string) error
	IsNormalCloseFunc    func(err error) bool
	RemoteAddrFunc       func() net.Addr
}

func (m *mockConn) ReadMessage(msg *message.Message) error {
	return m.ReadMessageFunc(msg)
}

func (m *mockConn) WriteMessage(msg message.Message) error {
	return m.WriteMessageFunc(msg)
}

func (m *mockConn) SetReadDeadline(t time.Time) error {
	return m.SetReadDeadlineFunc(t)
}

func (m *mockConn) SetWriteDeadline(t time.Time) error {
	return m.SetWriteDeadlineFunc(t)
}

func (m *mockConn) SetPongHandler(h func(appData string) error) {
	m.SetPongHandlerFunc(h)
}

func (m *mockConn) Close() error {
	return m.CloseFunc()
}

func (m *mockConn) WritePing() error {
	return m.WritePingFunc()
}

func (m *mockConn) WriteClose(reason string) error {
	return m.WriteCloseFunc(reason)
}

func (m *mockConn) IsNormalClose(err error) bool {
	return m.IsNormalCloseFunc(err)
}

func (m *mockConn) RemoteAddr() net.Addr {
	return m.RemoteAddrFunc()
}

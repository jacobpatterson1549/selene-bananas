package socket

import (
	"io/ioutil"
	"log"
	"net"
	"reflect"
	"testing"
	"time"
)

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
	return string(a) + "_NETWORK"
}

func (a mockAddr) String() string {
	return string(a)
}

func TestNewSocket(t *testing.T) {
	testLog := log.New(ioutil.Discard, "test", log.LstdFlags)
	timeFunc := func() int64 { return 89 }
	conn0 := &mockConn{}
	newSocketTests := []struct {
		wantOk bool
		want   *Socket
		Conn
		Config
	}{
		{}, // no Log
		{ // no conn
			Config: Config{
				Log: testLog,
			},
		},
		{ // no TimeFunc
			Conn: conn0,
			Config: Config{
				Log: testLog,
			},
		},
		{ // bad ReadWait
			Conn: conn0,
			Config: Config{
				Log:      testLog,
				TimeFunc: timeFunc,
			},
		},
		{ // bad WriteWait
			Conn: conn0,
			Config: Config{
				Log:      testLog,
				TimeFunc: timeFunc,
				ReadWait: 2 * time.Hour,
			},
		},
		{ // bad PingPeriod
			Conn: conn0,
			Config: Config{
				Log:        testLog,
				TimeFunc:   timeFunc,
				ReadWait:   2 * time.Hour,
				WriteWait:  2 * time.Hour,
				PingPeriod: 1 * time.Hour,
			},
		},
		{ // bad IdlePeriod
			Conn: conn0,
			Config: Config{
				Log:        testLog,
				TimeFunc:   timeFunc,
				ReadWait:   2 * time.Hour,
				WriteWait:  2 * time.Hour,
				PingPeriod: 1 * time.Hour,
			},
		},
		{ // bad HTTPPingPeriod
			Conn: conn0,
			Config: Config{
				Log:        testLog,
				TimeFunc:   timeFunc,
				ReadWait:   2 * time.Hour,
				WriteWait:  2 * time.Hour,
				PingPeriod: 1 * time.Hour,
				IdlePeriod: 1 * time.Hour,
			},
		},
		{ // PingPeriod not less than ReadWait
			Conn: conn0,
			Config: Config{
				Log:            testLog,
				TimeFunc:       timeFunc,
				ReadWait:       1 * time.Hour,
				WriteWait:      2 * time.Hour,
				PingPeriod:     1 * time.Hour,
				IdlePeriod:     1 * time.Hour,
				HTTPPingPeriod: 15 * time.Hour,
			},
		},
		{ // ok
			Conn: conn0,
			Config: Config{
				Log:            testLog,
				TimeFunc:       timeFunc,
				ReadWait:       2 * time.Hour,
				WriteWait:      2 * time.Hour,
				PingPeriod:     1 * time.Hour,
				IdlePeriod:     1 * time.Hour,
				HTTPPingPeriod: 15 * time.Hour,
			},
			want: &Socket{
				Conn: conn0,
				Config: Config{
					Log:            testLog,
					ReadWait:       2 * time.Hour,
					WriteWait:      2 * time.Hour,
					PingPeriod:     1 * time.Hour,
					IdlePeriod:     1 * time.Hour,
					HTTPPingPeriod: 15 * time.Hour,
				},
			},
			wantOk: true,
		},
	}
	for i, test := range newSocketTests {
		got, err := test.Config.NewSocket(test.Conn)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		default:
			got.Config.TimeFunc = nil // funcs cannot be compared
			if !reflect.DeepEqual(test.want, got) {
				t.Errorf("Test %v: sockets not equal:\nwanted: %v\ngot:    %v", i, test.want, got)
			}
		}
	}
}

func TestSocketRun(t *testing.T) {
	// TODO
}

func TestSocketReadMessages(t *testing.T) {
	// TODO
}

func TestSocketWriteMessages(t *testing.T) {
	// TODO
}

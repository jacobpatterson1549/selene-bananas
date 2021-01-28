package socket

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/game/player"
)

type (
	MockHijacker struct {
		http.ResponseWriter
		net.Conn
		*bufio.ReadWriter
	}

	RedirectConn struct {
		net.Conn
		io.Writer
	}
)

func (h MockHijacker) Header() http.Header {
	return h.ResponseWriter.Header()
}

func (h MockHijacker) Write(p []byte) (int, error) {
	return h.ReadWriter.Write(p)
}

func (h MockHijacker) WriteHeader(statusCode int) {
	h.ResponseWriter.WriteHeader(statusCode)
}

func (h MockHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return h.Conn, h.ReadWriter, nil
}

func (w RedirectConn) Write(p []byte) (int, error) {
	return w.Writer.Write(p)
}

func newSocketManager(t *testing.T, maxSockets int, maxPlayerSockets int) Manager {
	log := log.New(ioutil.Discard, "test", log.LstdFlags)
	timeFunc := func() int64 { return 21 }
	socketCfg := Config{
		Log:            log,
		Time:           timeFunc,
		ReadWait:       2 * time.Second,
		WriteWait:      1 * time.Second,
		PingPeriod:     1 * time.Second,
		IdlePeriod:     3 * time.Minute,
		HTTPPingPeriod: 1 * time.Minute,
	}
	managerCfg := ManagerConfig{
		Log:              log,
		MaxSockets:       maxSockets,
		MaxPlayerSockets: maxPlayerSockets,
		SocketConfig:     socketCfg,
	}
	m, err := managerCfg.NewManager()
	if err != nil {
		t.Fatalf("creating basic socket manager: %v", err)
	}
	return m
}

func newWebSocketResponse() http.ResponseWriter {
	w := httptest.NewRecorder()
	client, _ := net.Pipe()
	sr := strings.NewReader("reader")
	br := bufio.NewReader(sr)
	var bb bytes.Buffer
	bw := bufio.NewWriter(&bb)
	rw := bufio.NewReadWriter(br, bw)
	rc := RedirectConn{
		Conn:   client,
		Writer: bw,
	}
	h := MockHijacker{
		Conn:           rc,
		ReadWriter:     rw,
		ResponseWriter: w,
	}
	return &h
}

func newWebSocketRequest() *http.Request {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Add("Connection", "upgrade")
	r.Header.Add("Upgrade", "websocket")
	r.Header.Add("Sec-Websocket-Version", "13")
	r.Header.Add("Sec-WebSocket-Key", "3D8mi1hwk11RYYWU8rsdIg==")
	return r
}

func mockConnection(playerName string) (player.Name, http.ResponseWriter, *http.Request) {
	pn := player.Name(playerName)
	w := newWebSocketResponse()
	r := newWebSocketRequest()
	return pn, w, r
}

func TestManagerConfigValidate(t *testing.T) {
	testLog := log.New(ioutil.Discard, "test", log.LstdFlags)
	managerConfigValidateTests := []struct {
		Log              *log.Logger
		MaxSockets       int
		MaxPlayerSockets int
		wantOk           bool
	}{
		{},
		{ // no log
			MaxSockets:       1,
			MaxPlayerSockets: 1,
		},
		{ // low maxSockets
			Log:              testLog,
			MaxPlayerSockets: 1,
		},
		{ // low maxPlayerSockets
			Log:        testLog,
			MaxSockets: 1,
		},
		{ // maxSockets < maxPlayerSockets
			Log:              testLog,
			MaxSockets:       1,
			MaxPlayerSockets: 2,
		},
		{
			Log:              testLog,
			MaxSockets:       10,
			MaxPlayerSockets: 3,
			wantOk:           true,
		},
	}
	for i, test := range managerConfigValidateTests {
		cfg := ManagerConfig{
			Log:              test.Log,
			MaxSockets:       test.MaxSockets,
			MaxPlayerSockets: test.MaxPlayerSockets,
			// Not setting SocketConfig here because it is checked every time a socket is created
		}
		err := cfg.validate()
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: wanted error", i)
		}
	}
}

func TestAddSocket(t *testing.T) {
	m := newSocketManager(t, 1, 1)
	pn, w, r := mockConnection("selene")
	err := m.AddSocket(pn, w, r)
	if err != nil {
		t.Fatalf("unwanted error adding socket: %v", err)
	}
	gwsm := m.(*gorillaWebSocketManager)
	n := gwsm.numSockets()
	switch {
	case err != nil:
		t.Errorf("problem adding socket: %v", err)
	case n != 1:
		t.Errorf("wanted 1 socket, got %v", n)
	}
}

func TestAddSocketBadConnUpgrade(t *testing.T) {
	m := newSocketManager(t, 1, 1)
	pn, w, r := mockConnection("selene")
	r.Method = "POST"
	err := m.AddSocket(pn, w, r)
	if err == nil {
		t.Error("wanted error creating socket with bad request")
	}
}

func TestAddSocketBadSocketConfig(t *testing.T) {
	m := newSocketManager(t, 1, 1)
	m.(*gorillaWebSocketManager).Config.SocketConfig.Time = nil
	pn, w, r := mockConnection("selene")
	err := m.AddSocket(pn, w, r)
	if err == nil {
		t.Error("wanted error creating socket with bad config")
	}
}

func TestAddSocketMax(t *testing.T) {
	m := newSocketManager(t, 1, 1)
	pn, w, r := mockConnection("selene")
	err := m.AddSocket(pn, w, r)
	if err != nil {
		t.Fatalf("unwanted error adding socket: %v", err)
	}
	pn, w, r = mockConnection("selene")
	err = m.AddSocket(pn, w, r) // only one socket allowed
	if err == nil {
		t.Errorf("wanted error adding socket when maxSockets reached")
	}
}

func TestAddSocketTwo(t *testing.T) {
	m := newSocketManager(t, 2, 2)
	pn1, w, r := mockConnection("fred")
	err := m.AddSocket(pn1, w, r)
	if err != nil {
		t.Fatalf("unwanted error adding socket: %v", err)
	}
	pn2, w, r := mockConnection("barney")
	err = m.AddSocket(pn2, w, r)
	gwsm := m.(*gorillaWebSocketManager)
	switch {
	case err != nil:
		t.Errorf("problem adding socket: %v", err)
	case len(gwsm.playerSockets) != 2:
		t.Errorf("wanted 2 players to have a socket, got %v", len(gwsm.playerSockets))
	case len(gwsm.playerSockets[pn1]) != 1:
		t.Errorf("wanted 1 socket for %v, got %v", pn1, len(gwsm.playerSockets[pn1]))
	case len(gwsm.playerSockets[pn2]) != 1:
		t.Errorf("wanted 1 socket for %v, got %v", pn2, len(gwsm.playerSockets[pn2]))
	}
}

func TestAddSocketTwoSamePlayer(t *testing.T) {
	m := newSocketManager(t, 2, 1)
	pn, w, r := mockConnection("fred")
	err := m.AddSocket(pn, w, r)
	if err != nil {
		t.Fatalf("unwanted error adding socket: %v", err)
	}
	pn, w, r = mockConnection("fred")
	err = m.AddSocket(pn, w, r)
	if err == nil {
		t.Errorf("wanted error adding socket for same player over maxPlayerSockets")
	}
}

// func TestJoinGame(t *testing.T) {
// 	m := newSocketManager(t, 1, 1)
// 	s := mockSocket()
// }
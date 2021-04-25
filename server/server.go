// Package server runs the http server with allows users to open websockets to play the game
package server

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"sync"

	"github.com/jacobpatterson1549/selene-bananas/server/log"
)

type (
	// Server runs the site
	Server struct {
		Config
		wg          sync.WaitGroup
		log         log.Logger
		lobby       Lobby
		HTTPServer  *http.Server
		HTTPSServer *http.Server
	}

	// Tokenizer creates and reads tokens from http traffic.
	Tokenizer interface {
		Create(username string, points int) (string, error)
		ReadUsername(tokenString string) (string, error)
	}

	// Lobby is the place users can create, join, and participate in games.
	Lobby interface {
		Run(ctx context.Context, wg *sync.WaitGroup)
		AddUser(username string, w http.ResponseWriter, r *http.Request) error
		RemoveUser(username string)
	}
)

// Run the server asynchronously until it receives a shutdown signal.
// When the HTTP/HTTPS servers stop, errors are logged to the error channel.
func (s *Server) Run(ctx context.Context) <-chan error {
	errC := make(chan error, 2)
	s.runHTTPServer(ctx, errC)
	s.runHTTPSServer(ctx, errC)
	return errC
}

// runHTTPSServer runs the http server asynchronously, adding the return error to the channel when done.
// The server is only run if the HTTP address is valid.
func (s *Server) runHTTPServer(ctx context.Context, errC chan<- error) {
	if !s.validHTTPAddr() {
		return
	}
	go s.serveTCP(s.HTTPServer, errC, false, s.log)
}

// runHTTPSServer runs the https server in regards to the configuration, adding the return error to the channel when done.
func (s *Server) runHTTPSServer(ctx context.Context, errC chan<- error) {
	ctx, cancelFunc := context.WithCancel(ctx)
	s.lobby.Run(ctx, &s.wg)
	s.HTTPSServer.RegisterOnShutdown(cancelFunc)
	s.logServerStart()
	go s.serveTCP(s.HTTPSServer, errC, true, s.log)
}

func (s *Server) logServerStart() {
	var serverStartInfo string
	switch {
	case s.validHTTPAddr():
		serverStartInfo = "starting server at at https://127.0.0.1%v"
	default:
		serverStartInfo = "starting server at at http://127.0.0.1%v"
	}
	s.log.Printf(serverStartInfo, s.HTTPSServer.Addr)
}

// serveTCP runs the specified server on the TCP network, configuring tls if necessary.
// ServeTCP is closely derived from https://golang.org/src/net/http/server.go to allow key bytes rather than files
func (s *Server) serveTCP(svr *http.Server, errC chan<- error, tls bool, log log.Logger) (err error) {
	defer func() { errC <- err }()
	ln, err := net.Listen("tcp", svr.Addr)
	if err != nil {
		return err
	}
	defer ln.Close()
	switch {
	case !tls:
		// NOOP
	case s.validHTTPAddr():
		ln, err = s.tlsListener(ln)
		if err != nil {
			return err
		}
	case len(s.TLSCertPEM) != 0, len(s.TLSKeyPEM) != 0:
		log.Printf("Ignoring certificate since PORT was specified, using automated certificate management.")
	}
	err = svr.Serve(ln) // BLOCKING
	return
}

// tlsListener creates a TLS listener, wrapping the base listener.
// TLSListener is derived from https://golang.org/src/net/http/server.go to create a TLS Listener using the key bytes rather than files.
func (s *Server) tlsListener(l net.Listener) (net.Listener, error) {
	certificate, err := tls.X509KeyPair([]byte(s.TLSCertPEM), []byte(s.TLSKeyPEM))
	if err != nil {
		return nil, err
	}
	tlsCfg := &tls.Config{}
	tlsCfg.NextProtos = []string{"http/1.1"}
	tlsCfg.Certificates = []tls.Certificate{certificate}
	tlsListener := tls.NewListener(l, tlsCfg)
	return tlsListener, nil
}

// Shutdown asks the servers to shutdown and waits for the shutdown to complete.
// An error is returned if the server context times out.
func (s *Server) Shutdown(ctx context.Context) error {
	ctx, cancelFunc := context.WithTimeout(ctx, s.StopDur)
	defer cancelFunc()
	httpsShutdownErr := s.HTTPSServer.Shutdown(ctx)
	httpShutdownErr := s.HTTPServer.Shutdown(ctx)
	switch {
	case httpsShutdownErr != nil:
		return httpsShutdownErr
	case httpShutdownErr != nil:
		return httpShutdownErr
	}
	s.wg.Wait()
	return nil
}

package server

import (
	"net"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/server/log/logtest"
)

func TestTLSListener(t *testing.T) {
	// test certificates copied from example at https://golang.org/pkg/crypto/tls/#X509KeyPair
	certPem := `-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d
7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B
5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr
BgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1
NDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l
Wf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc
6MF9+Yw1Yy0t
-----END CERTIFICATE-----`
	keyPem := `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIIrYSSNQFaA2Hwf1duRSxKtLYX5CB04fSeQ6tF1aY/PuoAoGCCqGSM49
AwEHoUQDQgAEPR3tU2Fta9ktY+6P9G0cWO+0kETA6SFs38GecTyudlHz6xvCdz8q
EKTcWGekdmdDPsHloRNtsiCa697B2O9IFA==
-----END EC PRIVATE KEY-----`
	tlsListenerTests := []struct {
		Config
		net.Addr
		wantOk bool
	}{
		{ // bad config
		},
		{ // ok key pair
			Config: Config{
				TLSCertPEM: certPem,
				TLSKeyPEM:  keyPem,
			},
			wantOk: true,
		},
	}
	for i, test := range tlsListenerTests {
		testAddr := mockAddr("selene.pc")
		innerListener := mockListener{
			AddrFunc: func() net.Addr {
				return testAddr
			},
		}
		s := Server{
			Config: test.Config,
		}
		got, err := s.tlsListener(innerListener)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case reflect.DeepEqual(innerListener, got):
			t.Errorf("Test %v: wanted TLS listener to be different from innerListener: got %v", i, got)
		case !reflect.DeepEqual(testAddr, got.Addr()):
			t.Errorf("Test %v: listener addresses not equal: wanted %v, got %v", i, testAddr, got.Addr())
		}
	}
}

func TestLogServerStart(t *testing.T) {
	logServerStartTests := []struct {
		HTTPPort    int
		wantLogPart string
	}{
		{
			HTTPPort:    80,
			wantLogPart: "https://",
		},
		{
			wantLogPart: "http://",
		},
	}
	for i, test := range logServerStartTests {
		log := logtest.NewLogger()
		cfg := Config{
			HTTPPort: test.HTTPPort,
		}
		s := Server{
			log:         log,
			HTTPSServer: new(http.Server),
			Config:      cfg,
		}
		s.logServerStart()
		gotLog := log.String()
		if !strings.Contains(gotLog, test.wantLogPart) {
			t.Errorf("Test %v: wanted log to contain '%v', got '%v'", i, test.wantLogPart, gotLog)
		}
	}
}

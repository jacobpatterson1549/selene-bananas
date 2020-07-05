package server

import (
	"fmt"
	"net/http"
)

type (
	// Challenge token and key used to get a TLS certificate using the ACME HTTP-01
	Challenge struct {
		Token string
		Key   string
	}
)

const (
	// acmeHeader is the path of the endpoint to serve the token.key at
	acmeHeader = "/.well-known/acme-challenge/"
)

// isFor determines if a path is a request for an acme Challenge.  The challenge token must not be empty.
func (Challenge) isFor(path string) bool {
	return len(path) > len(acmeHeader) && path[:len(acmeHeader)] == acmeHeader
}

// handle writes the challenge to the response.
// The concatenation of the token, a peroid, and the key.
// The url of the request is not validated.
func (c Challenge) handle(w http.ResponseWriter, r *http.Request) error {
	if !c.isFor(r.URL.Path) || r.URL.Path[len(acmeHeader):] != c.Token {
		return fmt.Errorf("path '%v' is not for challenge", r.URL.Path)
	}
	data := c.Token + "." + c.Key
	if _, err := w.Write([]byte(data)); err != nil {
		return fmt.Errorf("writing acme token: %w", err)
	}
	return nil
}

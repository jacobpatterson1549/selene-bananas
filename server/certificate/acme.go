// Package certificate contains code related to managing Transport Layer Security for HTTPS connections
package certificate

import (
	"fmt"
	"io"
)

type (
	// Challenge token and key used to get a TLS certificate using the ACME HTTP-01
	Challenge struct {
		Token string
		Key   string
	}
)

const (
	// acmeHeader is the path of the endpoint to serve the challenge at.
	acmeHeader = "/.well-known/acme-challenge/"
)

// IsFor determines if a path is a request for an acme Challenge.  The challenge token must not be empty.
func (Challenge) IsFor(path string) bool {
	return len(path) > len(acmeHeader) && path[:len(acmeHeader)] == acmeHeader
}

// Handle writes the challenge to the response.
// Writes the concatenation of the token, a period, and the key.
func (c Challenge) Handle(w io.Writer, path string) error {
	if !c.IsFor(path) || path[len(acmeHeader):] != c.Token {
		return fmt.Errorf("path '%v' is not for challenge", path)
	}
	data := c.Token + "." + c.Key
	if _, err := w.Write([]byte(data)); err != nil {
		return fmt.Errorf("writing acme token: %w", err)
	}
	return nil
}

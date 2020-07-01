package server

import (
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

func (c Challenge) isFor(path string) bool {
	return len(c.Token) > 0 &&
		len(path) == len(acmeHeader)+len(c.Token) &&
		path[:len(acmeHeader)] == acmeHeader &&
		path[len(acmeHeader):] == c.Token
}

// handle writes the challenge to the response.
// The concatenation of the token, a peroid, and the key.
// The url of the request is not validated.
func (c Challenge) handle(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(c.Token + "." + c.Key))
}

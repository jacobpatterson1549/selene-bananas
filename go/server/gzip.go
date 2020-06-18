package server

import (
	"io"
	"net/http"
)

type (
	gzipResponseWriter struct {
		io.Writer
		http.ResponseWriter
	}
)

func (gzrw gzipResponseWriter) Write(p []byte) (n int, err error) {
	return gzrw.Writer.Write(p)
}

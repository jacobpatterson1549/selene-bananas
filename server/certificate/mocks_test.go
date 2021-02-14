package certificate

import "io"

type mockErrorWriter struct {
	writeErr error
	io.Writer
}

func (w mockErrorWriter) Write(p []byte) (n int, err error) {
	if w.writeErr != nil {
		return 0, w.writeErr
	}
	return w.Writer.Write(p)
}

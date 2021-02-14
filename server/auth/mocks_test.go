package auth

type mockErrorReader struct {
	readErr error
}

func (r mockErrorReader) Read(p []byte) (n int, err error) {
	return 0, r.readErr
}

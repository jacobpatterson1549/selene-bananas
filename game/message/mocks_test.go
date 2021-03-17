package message

// mockAddr implements the net.Addr interface
type mockAddr string

func (m mockAddr) Network() string {
	return string(m) + "_NETWORK"
}

func (m mockAddr) String() string {
	return string(m)
}

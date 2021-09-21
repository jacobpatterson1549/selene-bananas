package lobby

import "syscall/js"

type mockSocket struct {
	connectFunc func(event js.Value) error
	closeFunc   func()
}

func (m mockSocket) Connect(event js.Value) error {
	return m.connectFunc(event)
}

func (m mockSocket) Close() {
	m.closeFunc()
}

type mockLog struct {
	errorFunc func(text string)
}

func (m mockLog) Error(text string) {
	m.errorFunc(text)
}

type mockGame struct {
	leaveFunc func()
}

func (m mockGame) Leave() {
	m.leaveFunc()
}

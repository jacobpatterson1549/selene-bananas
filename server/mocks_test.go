package server

import (
	"context"
	"net"
	"net/http"
	"sync"

	"github.com/jacobpatterson1549/selene-bananas/db/user"
)

type mockTokenizer struct {
	CreateFunc       func(username string, points int) (string, error)
	ReadUsernameFunc func(tokenString string) (string, error)
}

func (m mockTokenizer) Create(username string, points int) (string, error) {
	return m.CreateFunc(username, points)
}

func (m mockTokenizer) ReadUsername(tokenString string) (string, error) {
	return m.ReadUsernameFunc(tokenString)
}

type mockUserDao struct {
	createFunc         func(ctx context.Context, u user.User) error
	loginFunc          func(ctx context.Context, u user.User) (*user.User, error)
	updatePasswordFunc func(ctx context.Context, u user.User, newP string) error
	deleteFunc         func(ctx context.Context, u user.User) error
	backendFunc        func() user.Backend
}

func (m mockUserDao) Create(ctx context.Context, u user.User) error {
	return m.createFunc(ctx, u)
}

func (m mockUserDao) Login(ctx context.Context, u user.User) (*user.User, error) {
	return m.loginFunc(ctx, u)
}

func (m mockUserDao) UpdatePassword(ctx context.Context, u user.User, newP string) error {
	return m.updatePasswordFunc(ctx, u, newP)
}

func (m mockUserDao) Delete(ctx context.Context, u user.User) error {
	return m.deleteFunc(ctx, u)
}

func (m mockUserDao) Backend() user.Backend {
	return m.backendFunc()
}

type mockLobby struct {
	runFunc        func(ctx context.Context, wg *sync.WaitGroup)
	addUserFunc    func(username string, w http.ResponseWriter, r *http.Request) error
	removeUserFunc func(username string)
}

func (m mockLobby) Run(ctx context.Context, wg *sync.WaitGroup) {
	m.runFunc(ctx, wg)
}

func (m mockLobby) AddUser(username string, w http.ResponseWriter, r *http.Request) error {
	return m.addUserFunc(username, w, r)
}

func (m mockLobby) RemoveUser(username string) {
	m.removeUserFunc(username)
}

// mockAddr implements the net.Addr interface
type mockAddr string

func (m mockAddr) Network() string {
	return string(m) + "_NETWORK"
}

func (m mockAddr) String() string {
	return string(m)
}

// mockListener implements the net.Listener interface
type mockListener struct {
	AcceptFunc func() (net.Conn, error)
	CloseFunc  func() error
	AddrFunc   func() net.Addr
}

func (m mockListener) Accept() (net.Conn, error) {
	return m.AcceptFunc()
}
func (m mockListener) Close() error {
	return m.CloseFunc()
}
func (m mockListener) Addr() net.Addr {
	return m.AddrFunc()
}

package server

import (
	"bytes"
	"context"
	"net/http"
	"sync"

	"github.com/jacobpatterson1549/selene-bananas/db/user"
)

type mockTokenizer struct {
	CreateFunc       func(username string, points int) (string, error)
	ReadUsernameFunc func(tokenString string) (string, error)
}

func (t mockTokenizer) Create(username string, points int) (string, error) {
	return t.CreateFunc(username, points)
}

func (t mockTokenizer) ReadUsername(tokenString string) (string, error) {
	return t.ReadUsernameFunc(tokenString)
}

type mockUserDao struct {
	createFunc         func(ctx context.Context, u user.User) error
	readFunc           func(ctx context.Context, u user.User) (*user.User, error)
	updatePasswordFunc func(ctx context.Context, u user.User, newP string) error
	deleteFunc         func(ctx context.Context, u user.User) error
}

func (ud mockUserDao) Create(ctx context.Context, u user.User) error {
	return ud.createFunc(ctx, u)
}

func (ud mockUserDao) Read(ctx context.Context, u user.User) (*user.User, error) {
	return ud.readFunc(ctx, u)
}

func (ud mockUserDao) UpdatePassword(ctx context.Context, u user.User, newP string) error {
	return ud.updatePasswordFunc(ctx, u, newP)
}

func (ud mockUserDao) Delete(ctx context.Context, u user.User) error {
	return ud.deleteFunc(ctx, u)
}

type mockLobby struct {
	runFunc        func(ctx context.Context, wg *sync.WaitGroup)
	addUserFunc    func(username string, w http.ResponseWriter, r *http.Request) error
	removeUserFunc func(username string)
}

func (l mockLobby) Run(ctx context.Context, wg *sync.WaitGroup) {
	l.runFunc(ctx, wg)
}

func (l mockLobby) AddUser(username string, w http.ResponseWriter, r *http.Request) error {
	return l.addUserFunc(username, w, r)
}

func (l mockLobby) RemoveUser(username string) {
	l.removeUserFunc(username)
}

// mockResponseWriter is more simple than httptest.ResponseRecorder.
// It implements http.ResponseWriter, logs everything written to its buffer, and stores the status code.
type mockResponseWriter struct {
	HTTPHeader http.Header
	bytes.Buffer
	StatusCode int
}

func (w *mockResponseWriter) Header() http.Header {
	return w.HTTPHeader
}

func (w *mockResponseWriter) Write(p []byte) (int, error) {
	return w.Buffer.Write(p)
}

func (w *mockResponseWriter) WriteHeader(statusCode int) {
	w.StatusCode = statusCode
}

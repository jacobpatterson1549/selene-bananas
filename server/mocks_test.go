package server

import (
	"context"
	"net/http"

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
	runFunc        func(ctx context.Context)
	addUserFunc    func(username string, w http.ResponseWriter, r *http.Request) error
	removeUserFunc func(username string)
}

func (l mockLobby) Run(ctx context.Context) {
	l.runFunc(ctx)
}

func (l mockLobby) AddUser(username string, w http.ResponseWriter, r *http.Request) error {
	return l.addUserFunc(username, w, r)
}

func (l mockLobby) RemoveUser(username string) {
	l.removeUserFunc(username)
}

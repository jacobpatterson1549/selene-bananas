package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/db/user"
)

func TestHandleUserCreate(t *testing.T) {
	handleUserCreateTests := []struct {
		username        string
		password        string
		wantHandleError bool
		daoErr          error
		wantCode        int
	}{
		{
			wantHandleError: true,
			wantCode:        500,
		},
		{
			username: "selene",
			password: "password123",
			daoErr:   fmt.Errorf("problem creating user"),
			wantCode: 500,
		},
		{
			username: "selene",
			password: "password123",
			wantCode: 200,
		},
	}
	for i, test := range handleUserCreateTests {
		s := Server{
			log: log.New(io.Discard, "", 0),
			userDao: mockUserDao{
				createFunc: func(ctx context.Context, u user.User) error {
					switch {
					case test.username != u.Username:
						t.Errorf("Test %v wanted username to update to be %v, got %v", i, test.username, u.Username)
					}
					return test.daoErr
				},
			},
		}
		r := httptest.NewRequest("", "/", nil)
		r.Form = make(url.Values)
		r.Form.Add("username", test.username)
		r.Form.Add("password_confirm", test.password)
		w := httptest.NewRecorder()
		s.handleUserCreate(w, r)
		gotCode := w.Code
		switch {
		case test.wantCode != gotCode:
			t.Errorf("Test %v: response codes not equal: wanted: %v, got: %v", i, test.wantCode, gotCode)
		case w.Body.Len() == 0:
			if gotCode != 200 {
				t.Errorf("Test %v: response body should not be empty", i)
			}
		}
	}
}

func TestHandleUserLogin(t *testing.T) {
	handleUserLoginTests := []struct {
		username     string
		password     string
		daoErr       error
		tokenizerErr error
		wantCode     int
	}{
		{
			wantCode: 500,
		},
		{
			username: "selene",
			password: "password123",
			daoErr:   fmt.Errorf("problem signing user in"),
			wantCode: 500,
		},
		{
			username: "eve",
			password: "l3tMeIn!",
			daoErr:   user.ErrIncorrectLogin,
			wantCode: 401,
		},
		{
			username:     "selene",
			password:     "password123",
			tokenizerErr: fmt.Errorf("problem creating token"),
			wantCode:     500,
		},
		{
			username: "selene",
			password: "password123",
			wantCode: 200,
		},
	}
	wantPoints := 8
	wantToken := "created token for logged-in user"
	for i, test := range handleUserLoginTests {
		var buf bytes.Buffer
		s := Server{
			log: log.New(&buf, "", 0),
			userDao: mockUserDao{
				loginFunc: func(ctx context.Context, u user.User) (*user.User, error) {
					switch {
					case test.username != u.Username:
						t.Errorf("Test %v wanted username to update to be %v, got %v", i, test.username, u.Username)
					case test.daoErr != nil:
						return nil, test.daoErr
					}
					u2 := user.User{
						Username: u.Username,
						Points:   wantPoints,
					}
					return &u2, nil
				},
			},
			tokenizer: mockTokenizer{
				CreateFunc: func(username string, points int) (string, error) {
					switch {
					case test.username != username:
						t.Errorf("Test %v wanted username to create token for to be %v, got %v", i, test.username, username)
					case wantPoints != points:
						t.Errorf("Test %v wanted points be %v, got %v", i, wantPoints, points)
					case test.tokenizerErr != nil:
						return "", test.tokenizerErr
					}
					return wantToken, nil
				},
			},
		}
		r := httptest.NewRequest("", "/", nil)
		r.Form = make(url.Values)
		r.Form.Add("username", test.username)
		r.Form.Add("password", test.password)
		w := httptest.NewRecorder()
		s.handleUserLogin(w, r)
		gotCode := w.Code
		switch {
		case test.wantCode != gotCode:
			t.Errorf("Test %v: response codes not equal: wanted: %v, got: %v", i, test.wantCode, gotCode)
		case w.Body.Len() == 0:
			t.Errorf("Test %v: response body should not be empty", i)
		}
	}
}

func TestHandleUserLobby(t *testing.T) {
	wantAccessToken := "selene_access_token"
	wantUsername := "selene"
	handleUserLobbyTests := []struct {
		accessToken string
		addUserErr  error
		wantCode    int
	}{
		{
			accessToken: "alice",
			wantCode:    401,
		},
		{
			accessToken: wantAccessToken,
			addUserErr:  fmt.Errorf("problem adding user"),
			wantCode:    500,
		},
		{
			accessToken: wantAccessToken,
			wantCode:    200,
		},
	}
	for i, test := range handleUserLobbyTests {
		s := Server{
			log: log.New(io.Discard, "", 0),
			tokenizer: mockTokenizer{
				ReadUsernameFunc: func(tokenString string) (string, error) {
					if test.accessToken != tokenString {
						t.Errorf("Test %v wanted tokenString to be %v, got %v", i, test.accessToken, tokenString)
					}
					if wantAccessToken != tokenString {
						return "", fmt.Errorf("problem reading access token")
					}
					return wantUsername, nil
				},
			},
			lobby: mockLobby{
				addUserFunc: func(username string, w http.ResponseWriter, r *http.Request) error {
					if username != wantUsername {
						return fmt.Errorf("wanted username %v, got %v", wantUsername, username)
					}
					return test.addUserErr
				},
			},
		}
		r := httptest.NewRequest("", "/", nil)
		r.Form = make(url.Values)
		r.Form.Add("access_token", test.accessToken)
		w := httptest.NewRecorder()
		s.handleUserLobby(w, r)
		gotCode := w.Code
		switch {
		case test.wantCode != gotCode:
			t.Errorf("Test %v: response codes not equal: wanted: %v, got: %v", i, test.wantCode, gotCode)
		case w.Body.Len() == 0:
			if gotCode != 200 { // connecting to the lobby will normally keep the connection open until the user leaves the lobby
				t.Errorf("Test %v: response body should not be empty", i)
			}
		}
	}
}

func TestHandleUserUpdatePassword(t *testing.T) {
	handleUserUpdatePasswordTests := []struct {
		username        string
		password        string
		newPassword     string
		daoUpdateErr    error
		wantCode        int
		wantLobbyRemove bool
	}{
		{
			username: "INVALID username!",
			wantCode: 500,
		},
		{
			username:     "selene",
			password:     "TOP_s3cret!",
			newPassword:  "MoR&_sCr3T",
			daoUpdateErr: fmt.Errorf("error updating user password"),
			wantCode:     500,
		},
		{
			username:        "selene",
			password:        "TOP_s3cret!",
			newPassword:     "MoR&_sCr3T",
			wantCode:        200,
			wantLobbyRemove: true,
		},
	}
	for i, test := range handleUserUpdatePasswordTests {
		userDao := mockUserDao{
			updatePasswordFunc: func(ctx context.Context, u user.User, newP string) error {
				switch {
				case test.username != u.Username:
					t.Errorf("Test %v: wanted user %v to be deleted, got %v", i, test.username, u.Username)
				case test.newPassword != newP:
					t.Errorf("Test %v: wanted new password to be %v, got %v", i, test.newPassword, newP)
				}
				return test.daoUpdateErr
			},
		}
		gotLobbyRemove := false
		l := mockLobby{
			removeUserFunc: func(username string) {
				if test.username != username {
					t.Errorf("Test %v: wanted %v to be removed from lobby, got %v", i, test.username, username)
				}
				gotLobbyRemove = true
			},
		}
		s := Server{
			log:     log.New(io.Discard, "", 0),
			userDao: userDao,
			lobby:   l,
		}
		r := httptest.NewRequest("", "/", nil)
		r.Form = make(url.Values)
		r.Form.Add("username", test.username)
		r.Form.Add("password", test.password)
		r.Form.Add("password_confirm", test.newPassword)
		w := httptest.NewRecorder()
		s.handleUserUpdatePassword(w, r)
		gotCode := w.Code
		switch {
		case test.wantCode != gotCode:
			t.Errorf("Test %v: response codes not equal: wanted: %v, got: %v", i, test.wantCode, gotCode)
		case w.Body.Len() == 0:
			if gotCode != 200 {
				t.Errorf("Test %v: response body should not be empty", i)
			}
		case test.wantLobbyRemove != gotLobbyRemove:
			t.Errorf("Test %v: wanted lobby.RemoveUser to be called %v, got %v", i, test.wantLobbyRemove, gotLobbyRemove)
		}
	}
}

func TestHandleUserDelete(t *testing.T) {
	handleUserDeleteTests := []struct {
		username        string
		password        string
		daoDeleteErr    error
		wantCode        int
		wantLobbyRemove bool
	}{
		{
			username: "INVALID username!",
			wantCode: 500,
		},
		{
			username:     "selene",
			password:     "TOP_s3cret!",
			daoDeleteErr: fmt.Errorf("error deleting user from dao"),
			wantCode:     500,
		},
		{
			username:        "selene",
			password:        "TOP_s3cret!",
			wantCode:        200,
			wantLobbyRemove: true,
		},
	}
	for i, test := range handleUserDeleteTests {
		userDao := mockUserDao{
			deleteFunc: func(ctx context.Context, u user.User) error {
				if test.username != u.Username {
					t.Errorf("Test %v: wanted user %v to be deleted, got %v", i, test.username, u.Username)
				}
				return test.daoDeleteErr
			},
		}
		gotLobbyRemove := false
		l := mockLobby{
			removeUserFunc: func(username string) {
				if test.username != username {
					t.Errorf("Test %v: wanted %v to be removed from lobby, got %v", i, test.username, username)
				}
				gotLobbyRemove = true
			},
		}
		s := Server{
			log:     log.New(io.Discard, "", 0),
			userDao: userDao,
			lobby:   l,
		}
		r := httptest.NewRequest("", "/", nil)
		r.Form = make(url.Values)
		r.Form.Add("username", test.username)
		r.Form.Add("password", test.password)
		w := httptest.NewRecorder()
		s.handleUserDelete(w, r)
		gotCode := w.Code
		switch {
		case test.wantCode != gotCode:
			t.Errorf("Test %v: response codes not equal: wanted: %v, got: %v", i, test.wantCode, gotCode)
		case w.Body.Len() == 0:
			if gotCode != 200 {
				t.Errorf("Test %v: response body should not be empty", i)
			}
		case test.wantLobbyRemove != gotLobbyRemove:
			t.Errorf("Test %v: wanted lobby.RemoveUser to be called %v, got %v", i, test.wantLobbyRemove, gotLobbyRemove)
		}
	}
}

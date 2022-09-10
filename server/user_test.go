package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/db/user"
	"github.com/jacobpatterson1549/selene-bananas/server/log/logtest"
)

func TestUserCreateHandler(t *testing.T) {
	userCreateHandlerTests := []struct {
		username        string
		password        string
		wantHandleError bool
		daoErr          error
		wantCode        int
	}{
		{
			username: "selene",
			password: "password123",
			daoErr:   fmt.Errorf("problem creating user (duplicate username or invalid username/password)"),
			wantCode: 500,
		},
		{
			username: "selene",
			password: "password123",
			wantCode: 200,
		},
	}
	for i, test := range userCreateHandlerTests {
		userDao := mockUserDao{
			createFunc: func(ctx context.Context, u user.User) error {
				switch {
				case test.username != u.Username:
					t.Errorf("Test %v wanted username to update to be %v, got %v", i, test.username, u.Username)
				}
				return test.daoErr
			},
		}
		log := logtest.DiscardLogger
		r := httptest.NewRequest("", "/", nil)
		r.Form = make(url.Values)
		r.Form.Add("username", test.username)
		r.Form.Add("password_confirm", test.password)
		w := httptest.NewRecorder()
		h := userCreateHandler(userDao, log)
		h.ServeHTTP(w, r)
		gotCode := w.Code
		switch {
		case test.wantCode != gotCode:
			t.Errorf("Test %v: response codes not equal after user create: wanted: %v, got: %v", i, test.wantCode, gotCode)
		case w.Body.Len() == 0:
			if gotCode != 200 {
				t.Errorf("Test %v: response body should not be empty after user create", i)
			}
		}
	}
}

func TestUserLoginHandler(t *testing.T) {
	userLoginHandlerTests := []struct {
		username     string
		password     string
		daoErr       error
		tokenizerErr error
		wantCode     int
		wantLog      bool
	}{
		{
			username: "selene",
			password: "password123",
			daoErr:   fmt.Errorf("problem signing user in"),
			wantCode: 500,
			wantLog:  true,
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
			wantLog:      true,
		},
		{
			username: "selene",
			password: "password123",
			wantCode: 200,
		},
	}
	wantPoints := 8
	wantToken := "created token for logged-in user"
	for i, test := range userLoginHandlerTests {
		userDao := mockUserDao{
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
		}
		tokenizer := mockTokenizer{
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
		}
		log := new(logtest.Logger)
		r := httptest.NewRequest("", "/", nil)
		r.Form = make(url.Values)
		r.Form.Add("username", test.username)
		r.Form.Add("password", test.password)
		w := httptest.NewRecorder()
		h := userLoginHandler(userDao, tokenizer, log)
		h.ServeHTTP(w, r)
		gotCode := w.Code
		gotLog := !log.Empty()
		switch {
		case test.wantCode != gotCode:
			t.Errorf("Test %v: response codes not equal after user login: wanted: %v, got: %v", i, test.wantCode, gotCode)
		case test.wantLog != gotLog:
			t.Errorf("Test %v: wanted or did not want log states not equal: wanted %v, got: %v - '%v'", i, test.wantLog, gotLog, log.String())
		}
	}
}

func TestUserLobbyConnectHandler(t *testing.T) {
	wantAccessToken := "selene_access_token"
	wantUsername := "selene"
	userLobbyConnectHandlerTests := []struct {
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
	for i, test := range userLobbyConnectHandlerTests {
		tokenizer := mockTokenizer{
			ReadUsernameFunc: func(tokenString string) (string, error) {
				if test.accessToken != tokenString {
					t.Errorf("Test %v wanted tokenString to be %v, got %v", i, test.accessToken, tokenString)
				}
				if wantAccessToken != tokenString {
					return "", fmt.Errorf("problem reading access token")
				}
				return wantUsername, nil
			},
		}
		lobby := mockLobby{
			addUserFunc: func(username string, w http.ResponseWriter, r *http.Request) error {
				if username != wantUsername {
					return fmt.Errorf("wanted username %v, got %v", wantUsername, username)
				}
				return test.addUserErr
			},
		}
		log := logtest.DiscardLogger
		r := httptest.NewRequest("", "/", nil)
		r.Form = make(url.Values)
		r.Form.Add("access_token", test.accessToken)
		w := httptest.NewRecorder()
		h := userLobbyConnectHandler(tokenizer, lobby, log)
		h.ServeHTTP(w, r)
		gotCode := w.Code
		switch {
		case test.wantCode != gotCode:
			t.Errorf("Test %v: response codes not equal after user lobby connect: wanted: %v, got: %v", i, test.wantCode, gotCode)
		case w.Body.Len() == 0:
			if gotCode != 200 { // connecting to the lobby will normally keep the connection open until the user leaves the lobby
				t.Errorf("Test %v: response body should not be empty after user lobby connect", i)
			}
		}
	}
}

func TestUserUpdatePasswordHandler(t *testing.T) {
	userUpdatePasswordHandlerTests := []struct {
		username        string
		password        string
		newPassword     string
		daoUpdateErr    error
		wantCode        int
		wantLobbyRemove bool
	}{
		{
			username:     "selene",
			password:     "TOP_s3cret!1",
			newPassword:  "MoR&_sCr3T1",
			daoUpdateErr: user.ErrIncorrectLogin,
			wantCode:     401,
		},
		{
			username:     "selene",
			password:     "TOP_s3cret!2",
			newPassword:  "MoR&_sCr3T2",
			daoUpdateErr: fmt.Errorf("error updating user password"),
			wantCode:     500,
		},
		{
			username:        "selene",
			password:        "TOP_s3cret!3",
			newPassword:     "MoR&_sCr3T3",
			wantCode:        200,
			wantLobbyRemove: true,
		},
	}
	for i, test := range userUpdatePasswordHandlerTests {
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
		lobby := mockLobby{
			removeUserFunc: func(username string) {
				if test.username != username {
					t.Errorf("Test %v: wanted %v to be removed from lobby, got %v", i, test.username, username)
				}
				gotLobbyRemove = true
			},
		}
		log := logtest.DiscardLogger
		r := httptest.NewRequest("", "/", nil)
		r.Form = make(url.Values)
		r.Form.Add("username", test.username)
		r.Form.Add("password", test.password)
		r.Form.Add("password_confirm", test.newPassword)
		w := httptest.NewRecorder()
		h := userUpdatePasswordHandler(userDao, lobby, log)
		h.ServeHTTP(w, r)
		gotCode := w.Code
		switch {
		case test.wantCode != gotCode:
			t.Errorf("Test %v: response codes not equal after user password update: wanted: %v, got: %v", i, test.wantCode, gotCode)
		case w.Body.Len() == 0:
			if gotCode != 200 {
				t.Errorf("Test %v: response body should not be empty after user password update", i)
			}
		case test.wantLobbyRemove != gotLobbyRemove:
			t.Errorf("Test %v: wanted lobby.RemoveUser to be called %v, got %v", i, test.wantLobbyRemove, gotLobbyRemove)
		}
	}
}

func TestUserDeleteHandler(t *testing.T) {
	userDeleteHandlerTests := []struct {
		username        string
		password        string
		daoDeleteErr    error
		wantCode        int
		wantLobbyRemove bool
	}{
		{
			username:     "selene",
			password:     "TOP_s3cret!i",
			daoDeleteErr: user.ErrIncorrectLogin,
			wantCode:     401,
		},
		{
			username:     "selene",
			password:     "TOP_s3cret!ii",
			daoDeleteErr: fmt.Errorf("error deleting user from dao"),
			wantCode:     500,
		},
		{
			username:        "selene",
			password:        "TOP_s3cret!iii",
			wantCode:        200,
			wantLobbyRemove: true,
		},
	}
	for i, test := range userDeleteHandlerTests {
		userDao := mockUserDao{
			deleteFunc: func(ctx context.Context, u user.User) error {
				if test.username != u.Username {
					t.Errorf("Test %v: wanted user %v to be deleted, got %v", i, test.username, u.Username)
				}
				return test.daoDeleteErr
			},
		}
		gotLobbyRemove := false
		lobby := mockLobby{
			removeUserFunc: func(username string) {
				if test.username != username {
					t.Errorf("Test %v: wanted %v to be removed from lobby, got %v", i, test.username, username)
				}
				gotLobbyRemove = true
			},
		}
		log := logtest.DiscardLogger
		r := httptest.NewRequest("", "/", nil)
		r.Form = make(url.Values)
		r.Form.Add("username", test.username)
		r.Form.Add("password", test.password)
		w := httptest.NewRecorder()
		h := userDeleteHandler(userDao, lobby, log)
		h.ServeHTTP(w, r)
		gotCode := w.Code
		switch {
		case test.wantCode != gotCode:
			t.Errorf("Test %v: response codes not equal after user delete: wanted: %v, got: %v", i, test.wantCode, gotCode)
		case w.Body.Len() == 0:
			if gotCode != 200 {
				t.Errorf("Test %v: response body should not be empty after user delete", i)
			}
		case test.wantLobbyRemove != gotLobbyRemove:
			t.Errorf("Test %v: wanted lobby.RemoveUser to be called %v, got %v", i, test.wantLobbyRemove, gotLobbyRemove)
		}
	}
}

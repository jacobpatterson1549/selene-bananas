package server

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/db/user"
)

func TestHandleUserCreate(t *testing.T) {
	handleUserCreateTests := []struct {
		username        string
		password        string
		wantHandleError bool
		daoErr          error
		wantOk          bool
	}{
		{
			wantHandleError: true,
		},
		{
			username: "selene",
			password: "password123",
			daoErr:   fmt.Errorf("problem creating user"),
		},
		{
			username: "selene",
			password: "password123",
			wantOk:   true,
		},
	}
	for i, test := range handleUserCreateTests {
		s := Server{
			log: log.New(ioutil.Discard, "test", log.LstdFlags),
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
		r := http.Request{
			Form: url.Values{
				"username":         {test.username},
				"password_confirm": {test.password},
			},
		}
		w := httptest.NewRecorder()
		s.handleUserCreate(w, &r)
		switch {
		case test.wantOk:
			want := 200
			got := w.Code
			if want != got {
				t.Errorf("Test %v: wanted response code to be %v, but was %v", i, want, got)
			}
		case test.daoErr != nil:
			want := test.daoErr.Error()
			got := w.Body.String()
			if !strings.Contains(got, want) {
				t.Errorf("Test %v: wanted response body to be %v, but was %v", i, want, got)
			}
			fallthrough
		default: // user creation error
			want := 500
			got := w.Code
			if want != got {
				t.Errorf("Test %v: wanted response code to be %v, but was %v", i, want, got)
			}
		}
	}
}

func TestHandleUserLogin(t *testing.T) {
	handleUserLoginTests := []struct {
		username        string
		password        string
		wantHandleError bool
		daoErr          error
		tokenizerErr    error
		wantCode        int
	}{
		{
			wantHandleError: true,
			wantCode:        500,
		},
		{
			username: "selene",
			password: "password123",
			daoErr:   fmt.Errorf("problem signing user in"),
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
	wantToken := "created token for logged user"
	for i, test := range handleUserLoginTests {
		s := Server{
			log: log.New(ioutil.Discard, "test", log.LstdFlags),
			userDao: mockUserDao{
				readFunc: func(ctx context.Context, u user.User) (*user.User, error) {
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
		r := http.Request{
			Form: url.Values{
				"username": {test.username},
				"password": {test.password},
			},
		}
		w := httptest.NewRecorder()
		s.handleUserLogin(w, &r)
		gotCode := w.Code
		if test.wantCode != gotCode {
			t.Errorf("Test %v: wanted response code to be %v, but was %v", i, test.wantCode, gotCode)
		}
		switch {
		case test.wantCode == 200:
			gotBody := w.Body.String()
			if wantToken != gotBody {
				t.Errorf("Test %v: wanted response body to be %v, but was %v", i, wantToken, gotBody)
			}
			continue
		case test.tokenizerErr != nil:
			want := test.tokenizerErr.Error()
			got := w.Body.String()
			// daoErr should be mapped to user-friendly message, do not checkit
			if !strings.Contains(got, want) {
				t.Errorf("Test %v: wanted response body to be %v, but was %v", i, want, got)
			}
		default:
			if w.Body.Len() == 0 {
				t.Errorf("Test %v: response body should not be empty when an error occurred", i)
			}
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
			log: log.New(ioutil.Discard, "test", log.LstdFlags),
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
		r := http.Request{
			Form: url.Values{
				"access_token": {test.accessToken},
			},
		}
		w := httptest.NewRecorder()
		s.handleUserLobby(w, &r)
		gotCode := w.Code
		if test.wantCode != gotCode {
			t.Errorf("Test %v: wanted response code to be %v, but was %v.  Response body: %v", i, test.wantCode, gotCode, w.Body.String())
		}
	}
}

func TestHandleUserUpdatePassword(t *testing.T) {
	handleUserUpdatePasswordTests := []struct {
		username        string
		password        string
		newPassword     string
		wantHandleError bool
		daoErr          error
		wantRemoveUser  bool
	}{
		{
			wantHandleError: true,
		},
		{
			username:    "selene",
			password:    "password123",
			newPassword: "top_s3cr3t!",
			daoErr:      fmt.Errorf("problem updating user password"),
		},
		{
			username:       "selene",
			password:       "password123",
			wantRemoveUser: true,
		},
	}
	for i, test := range handleUserUpdatePasswordTests {
		userRemoved := false
		s := Server{
			log: log.New(ioutil.Discard, "test", log.LstdFlags),
			userDao: mockUserDao{
				updatePasswordFunc: func(ctx context.Context, u user.User, newP string) error {
					switch {
					case test.username != u.Username:
						t.Errorf("Test %v wanted username to update to be %v, got %v", i, test.username, u.Username)
					case newP != test.newPassword:
						t.Errorf("Test %v wanted username to update to be %v, got %v", i, test.newPassword, newP)
					}
					return test.daoErr
				},
			},
			lobby: mockLobby{
				removeUserFunc: func(username string) {
					if test.username != username {
						t.Errorf("wanted username %v, got %v", test.username, username)
					}
					userRemoved = true
				},
			},
		}
		r := http.Request{
			Form: url.Values{
				"username":         {test.username},
				"password":         {test.password},
				"password_confirm": {test.newPassword},
			},
		}
		w := httptest.NewRecorder()
		s.handleUserUpdatePassword(w, &r)
		switch {
		case test.wantRemoveUser:
			if !userRemoved {
				t.Errorf("Test %v: wanted user to be removed, but was not", i)
			}
			want := 200
			got := w.Code
			if want != got {
				t.Errorf("Test %v: wanted response code to be %v, but was %v", i, want, got)
			}
		case test.daoErr != nil:
			want := test.daoErr.Error()
			got := w.Body.String()
			if !strings.Contains(got, want) {
				t.Errorf("Test %v: wanted response body to be %v, but was %v", i, want, got)
			}
			fallthrough
		default: // user creation error
			if userRemoved {
				t.Errorf("Test %v: wanted user to notbe removed, but was", i)
			}
			want := 500
			got := w.Code
			if want != got {
				t.Errorf("Test %v: wanted response code to be %v, but was %v", i, want, got)
			}
		}
	}
}

func TestHandleUserDelete(t *testing.T) {
	handleUserDeleteTests := []struct {
		username        string
		password        string
		wantHandleError bool
		daoErr          error
		wantRemoveUser  bool
	}{
		{
			wantHandleError: true,
		},
		{
			username: "selene",
			password: "password123",
			daoErr:   fmt.Errorf("problem updating user password"),
		},
		{
			username:       "selene",
			password:       "password123",
			wantRemoveUser: true,
		},
	}
	for i, test := range handleUserDeleteTests {
		userRemoved := false
		s := Server{
			log: log.New(ioutil.Discard, "test", log.LstdFlags),
			userDao: mockUserDao{
				deleteFunc: func(ctx context.Context, u user.User) error {
					switch {
					case test.username != u.Username:
						t.Errorf("Test %v wanted username to update to be %v, got %v", i, test.username, u.Username)
					}
					return test.daoErr
				},
			},
			lobby: mockLobby{
				removeUserFunc: func(username string) {
					if test.username != username {
						t.Errorf("wanted username %v, got %v", test.username, username)
					}
					userRemoved = true
				},
			},
		}
		r := http.Request{
			Form: url.Values{
				"username": {test.username},
				"password": {test.password},
			},
		}
		w := httptest.NewRecorder()
		s.handleUserDelete(w, &r)
		switch {
		case test.wantRemoveUser:
			if !userRemoved {
				t.Errorf("Test %v: wanted user to be removed, but was not", i)
			}
			want := 200
			got := w.Code
			if want != got {
				t.Errorf("Test %v: wanted response code to be %v, but was %v", i, want, got)
			}
		case test.daoErr != nil:
			want := test.daoErr.Error()
			got := w.Body.String()
			if !strings.Contains(got, want) {
				t.Errorf("Test %v: wanted response body to be %v, but was %v", i, want, got)
			}
			fallthrough
		default: // user creation error
			if userRemoved {
				t.Errorf("Test %v: wanted user to notbe removed, but was", i)
			}
			want := 500
			got := w.Code
			if want != got {
				t.Errorf("Test %v: wanted response code to be %v, but was %v", i, want, got)
			}
		}
	}
}

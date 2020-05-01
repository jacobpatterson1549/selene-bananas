// Package server runs the http server with allows users to open websockets to play the game
package server

import (
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
	"github.com/jacobpatterson1549/selene-bananas/go/server/game"
)

type (
	// Config contains fields which describe the server
	Config struct {
		AppName       string
		Port          string
		Database      db.Database
		Log           *log.Logger
		WordsFileName string
	}

	// Server can be run to serve the site
	Server interface {
		// Run starts the server
		Run() error
	}

	server struct {
		data              templateData
		addr              string
		log               *log.Logger
		handler           http.Handler
		staticFileHandler http.Handler
		upgrader          *websocket.Upgrader
		lobby             game.Lobby
		userDao           db.UserDao
		tokenizer         Tokenizer
	}

	templateData struct {
		ApplicationName string
	}
)

// NewServer creates a Server from the Config
func (cfg Config) NewServer() (Server, error) {
	data := templateData{
		ApplicationName: cfg.AppName,
	}
	addr := fmt.Sprintf(":%s", cfg.Port)
	serveMux := new(http.ServeMux)
	staticFileHandler := http.FileServer(http.Dir("./static"))
	rand := rand.New(rand.NewSource(time.Now().Unix()))
	tokenizer, err := newTokenizer(rand)
	if err != nil {
		cfg.Log.Fatal(err)
	}
	userDao := db.NewUserDao(cfg.Database)
	err = userDao.Setup()
	if err != nil {
		cfg.Log.Fatal(err)
	}
	lobby, err := game.NewLobby(cfg.Log, game.FileSystemWordsSupplier(cfg.WordsFileName), userDao, rand)
	if err != nil {
		cfg.Log.Fatal(err)
	}
	s := server{
		data:              data,
		addr:              addr,
		log:               cfg.Log,
		handler:           serveMux,
		staticFileHandler: staticFileHandler,
		lobby:             lobby,
		userDao:           userDao,
		tokenizer:         tokenizer,
	}
	serveMux.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("js"))))
	serveMux.HandleFunc("/", s.httpMethodHandler)
	return s, nil
}

func (s server) Run() error {
	httpServer := &http.Server{
		Addr:    s.addr,
		Handler: s.handler,
	}
	s.log.Println("starting server - locally running at http://127.0.0.1" + httpServer.Addr)
	err := httpServer.ListenAndServe() // BLOCKS
	if err != http.ErrServerClosed {
		return fmt.Errorf("server stopped unexpectedly: %w", err)
	}
	return nil
}

func (s server) httpMethodHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	switch r.Method {
	case "GET":
		err = s.httpGetHandler(w, r)
	case "POST":
		err = s.httpPostHandler(w, r)
	case "PUT":
	default:
		httpError(w, http.StatusMethodNotAllowed)
	}
	if err != nil {
		s.log.Printf("server error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s server) httpGetHandler(w http.ResponseWriter, r *http.Request) error {
	switch r.URL.Path {
	case "/":
		err := s.handleTemplate(w, r)
		if err != nil {
			return fmt.Errorf("rendering template: %w", err)
		}
	case "/user_join_lobby":
		err := r.ParseForm()
		if err != nil {
			return fmt.Errorf("parsing form: %w", err)
		}
		tokenString := r.FormValue("access_token")
		tokenUsername, err := s.tokenizer.Read(tokenString)
		if err != nil {
			httpError(w, http.StatusUnauthorized)
			return nil
		}
		return s.handleUserJoinLobby(w, r, tokenUsername)
	case "/user_logout", "/ping":
		_, err := s.checkAuthorization(r)
		if err != nil {
			httpError(w, http.StatusUnauthorized)
			return nil
		}
	default:
		s.staticFileHandler.ServeHTTP(w, r)
	}
	return nil
}

func (s server) httpPostHandler(w http.ResponseWriter, r *http.Request) error {
	var tokenUsername db.Username
	var err error
	switch r.URL.Path {
	case "/user_create", "/user_login":
	default:
		tokenUsername, err = s.checkAuthorization(r)
		if err != nil {
			httpError(w, http.StatusUnauthorized)
			return nil
		}
	}
	switch r.URL.Path {
	case "/user_create":
		err = s.handleUserCreate(w, r)
	case "/user_login":
		err = s.handleUserLogin(w, r)
	case "/user_update_password":
		err = s.handleUserUpdatePassword(w, r, tokenUsername)
	case "/user_delete":
		err = s.handleUserDelete(w, r, tokenUsername)
	default:
		httpError(w, http.StatusNotFound)
	}
	return err
}

func (s server) handleTemplate(w http.ResponseWriter, r *http.Request) error {
	t := template.New("main.html")
	templateFileGlobs := []string{
		"html/*.html",
		"html/**/*.html",
	}
	for _, g := range templateFileGlobs {
		_, err := t.ParseGlob(g)
		if err != nil {
			return err
		}
	}
	return t.Execute(w, s.data)
}

func httpError(w http.ResponseWriter, statusCode int) {
	http.Error(w, http.StatusText(statusCode), statusCode)
}

func (s server) addAuthorization(w http.ResponseWriter, u db.User) error {
	token, err := s.tokenizer.Create(u)
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(token))
	if err != nil {
		return fmt.Errorf("writing authorization token: %w", err)
	}
	return nil
}

func (s server) checkAuthorization(r *http.Request) (db.Username, error) {
	authorization := r.Header.Get("Authorization")
	if len(authorization) < 7 || authorization[:7] != "Bearer " {
		return "", fmt.Errorf("invalid authorization header: %v", authorization)
	}
	tokenString := authorization[7:]
	// TODO: make sure user modifications (update/delete/logout) are only done for this user
	return s.tokenizer.Read(tokenString)
}

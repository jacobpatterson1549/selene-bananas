# ![selene-bananas favicon](resources/favicon.ico) selene-bananas

[![Build Status](https://travis-ci.org/jacobpatterson1549/selene-bananas.svg?branch=master)](https://travis-ci.org/jacobpatterson1549/selene-bananas)
[![Go Report Card](https://goreportcard.com/badge/github.com/jacobpatterson1549/selene-bananas)](https://goreportcard.com/report/github.com/jacobpatterson1549/selene-bananas)
[![GoDoc](https://godoc.org/github.com/jacobpatterson1549/selene-bananas?status.svg)](https://godoc.org/github.com/jacobpatterson1549/selene-bananas)


## A Banagrams clone

A tile-based word-forming game based on the popular Banagrams game.  <https://bananagrams.com/games/bananagrams>

With WebSockets, users can play a word game together over a network.

Uses WebAssembly to manage browser logic.

## Dependencies

New dependencies are automatically added to [go.mod](go/go.mod) when the project is built.
* [pq](https://github.com/lib/pq) provides the Postgres driver for storing user passwords and points
* [Gorilla Websockets](https://github.com/gorilla/websocket) are used for bidirectional communication between users and the server
* [jwt-go](https://github.com/dgrijalva/jwt-go) is used for stateless web sessions
* [crypto](https://github.com/golang/crypto) is used to  encrypt passwords with bcrypt
* [Font-Awesome](https://github.com/FortAwesome/Font-Awesome) provides the "copyright", "github," and, "linkedin" icons on the about page; they were copied from version [5.13.0](https://github.com/FortAwesome/Font-Awesome/releases/tag/5.13.0) to [resources/fa](resources/fa).

## Build/Run

### Environment configuration

Environment variables are needed to customize the server.  Sample config:
```
DATABASE_URL=postgres://selene:selene123@127.0.0.1:54320/selene_bananas_db?sslmode=disable
WORDS_FILE=/usr/share/dict/american-english-large
```

It is recommended to install the [wamerican-large](https://packages.debian.org/buster/wamerican-large) package.  This package provides /usr/share/dict/american-english-large to be used as a words list in games.  Lowercase words are read from the word list for checking valid words in the game.  This can be overridden by providing the `WORDS_FILE` environment variable.

For development, set `CACHE_SECONDS` to `0` to not cache files.

### Database

The app stores user information in a Postgresql database.  When the app starts, files in the [resources/sql](resources/sql) folder are ran to ensure database objects functions are fresh.

#### localhost

A Postgresql database can be created with the command below.  Change the `PGUSER` and `PGPASSWORD` variables.  The command requires administrator access.
```bash
PGDATABASE="selene_bananas_db" \
PGUSER="selene" \
PGPASSWORD="selene123" \
PGHOSTADDR="127.0.0.1" \
PGPORT="5432" \
sh -c ' \
sudo -u postgres psql \
-c "CREATE DATABASE $PGDATABASE" \
-c "CREATE USER $PGUSER WITH ENCRYPTED PASSWORD '"'"'$PGPASSWORD'"'"'" \
-c "GRANT ALL PRIVILEGES ON DATABASE $PGDATABASE TO $PGUSER" \
&& echo DATABASE_URL=postgres://$PGUSER:$PGPASSWORD@$PGHOSTADDR:$PGPORT/$PGDATABASE'
```

### HTTPS

The app requires HTTP TLS (HTTPS) to run. Insecure http requests are redirected to https.

#### ACME

The server can verify its identity over http to pass a Automatic Certificate Management Environment (ACME) HTTP-01 challenge.  Add the `-acme-challenge-token` and `-acme-challenge-key` parameters with necessary values when running the server to return correct responses when the server's identity is challenged to create TLS certificates.  After the certificates are created, remove the acme-* flags, and specify the certificate and key with the `-tls-cert-file` and `-tls-key-file` flags. See [letsencrypt.org](https://letsencrypt.org/docs/challenge-types/#http-01-challenge) for more information about challenges.

#### localhost

Use [mkcert](https://github.com/FiloSottile/mkcert) to configure a development machine to accept local certificates.
```bash
go get github.com/FiloSottile/mkcert
mkcert -install
```
Generate certificates for localhost at 127.0.0.1
```bash
mkcert 127.0.0.1
```
Then, add the certificate files to the run environment configuration in `.env`.  The certificate files should be in the root of the application, but are aliased to be up a directory since the server runs in the build folder when running locally.
```
TLS_CERT_FILE=../127.0.0.1.pem
TLS_KEY_FILE=../127.0.0.1-key.pem
```

### Server Ports

By default, the server will run on ports 80 and 443 for http and https traffic.  All http traffic is redirected to https.  To override the ports, use the HTTP_PORT and HTTPS_PORT flags.

If the server handles HTTPS by providing its own certificate, use the `PORT` variable to specify the https port.  When POST is defined, no HTTP server will be started from `HTTP_PORT` and certificates are not read from the `TLS_CERT_FILE` and `TLS_KEY_FILE`.

##### Local Default TCP HTTP Ports

Run `make serve-tcp` to run on port 80 for HTTP and port 443 for HTTPS (default TCP ports).  Using these ports requires `sudo` (root) access.

### Make

The [Makefile](Makefile) runs the application locally.  This requires Go and a Postgres database to be installed.  [Node](https://github.com/nodejs) is needed to run WebAssembly tests.  Run `make serve` to build and run the application.

### Docker

Launching the application with [Docker](https://www.docker.com) requires minimal configuration.

1. Install [docker-compose](https://github.com/docker/compose)
1. Set database environment variables in the `.env` file in project root (next to Dockerfile).
    ```
    POSTGRES_DB=selene_bananas_db
    POSTGRES_USER=selene
    POSTGRES_PASSWORD=selene123
    POSTGRES_PORT=54320
    ```
1. Ensure the files for the `TLS_CERT_FILE` and `TLS_KEY_FILE` environment variables are located in the project folder and are **NOT** aliased relative to the build folder.  The variables should look like `TLS_CERT_FILE=127.0.0.1.pem`.
1. Run `docker-compose up` to launch the application.
1. Access application by opening <http://localhost:8000>.
# selene-bananas

[![Build Status](https://travis-ci.org/jacobpatterson1549/selene-bananas.svg?branch=master)](https://travis-ci.org/jacobpatterson1549/selene-bananas)
[![Go Report Card](https://goreportcard.com/badge/github.com/jacobpatterson1549/selene-bananas)](https://goreportcard.com/report/github.com/jacobpatterson1549/selene-bananas)
[![GoDoc](https://godoc.org/github.com/jacobpatterson1549/selene-bananas?status.svg)](https://godoc.org/github.com/jacobpatterson1549/selene-bananas)


## A Banagrams clone

A tile-based word-forming game based on the popular Banagrams game.  <https://bananagrams.com/games/bananagrams>

With WebSockets, users can play a word game together over a network.

Uses WebAssembly to manage browser logic.

## Dependencies

New dependencies are automatically added to [go.mod](go.mod) when the project is built.
* [pq](https://github.com/lib/pq) provides the Postgres driver for storing user passwords and points
* [Gorilla Websockets](https://github.com/gorilla/websocket) are used for bidirectional communication between users and the server
* [jwt-go](https://github.com/dgrijalva/jwt-go) is used for stateless web sessions
* [crypto](https://github.com/golang/crypto) is used to  encrypt passwords with bcrypt
* [Font-Awesome](https://github.com/FortAwesome/Font-Awesome) provides the "copyright", "github," "linkedin", and "gavel" icons on the about page; they were copied from version [5.13.0](https://github.com/FortAwesome/Font-Awesome/releases/tag/5.13.0) to [resources/fa](resources/fa).

## Build

Building the application requires [Go 1.16](https://golang.org/dl/).

The [Makefile](Makefile) builds and runs the application. Run `make` without any arguments to build the server with the client and other resources embedded in it.  This will likely need to be done before using an IDE in order to generate some files and prepropulate the embedded filesystem used by the the server.

[Node](https://github.com/nodejs) is needed to run WebAssembly tests.

Run `make serve` to build and run the application.

### Environment Configuration

Environment variables are needed to customize the server.  Sample config:
```
DATABASE_URL=postgres://selene:selene123@127.0.0.1:54320/selene_bananas_db?sslmode=disable
HTTP_PORT=8001
HTTPS_PORT=8000
```

It is recommended to install the [wamerican-large](https://packages.debian.org/buster/wamerican-large) package.  This package provides /usr/share/dict/american-english-large to be used as a words list in games.  Lowercase words are read from the word list for checking valid words in the game.  This can be overridden by providing the `WORDS_FILE` variable when running make: `make WORDS_FILE=/path/to/words/file.txt`.

For development, set `CACHE_SECONDS` to `0` to not cache static and template resources.

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

The app can be run on HTTP over TLS (HTTPS). If running on TLS, HTTP requests are redirected to HTTPS.

#### ACME

The server can verify its identity over HTTP to pass a Automatic Certificate Management Environment (ACME) HTTP-01 challenge.  Add the `--acme-challenge-token` and `--acme-challenge-key` parameters with necessary values when running the server to return correct responses when the server's identity is challenged to create TLS certificates.  After the certificates are created, remove the acme-* flags, and replace the [resources/tls-cert.pem](resources/tls-cert.pem) and [resources/tls-key.pem](resources/tls-key.pem) files with the certificates. See [letsencrypt.org](https://letsencrypt.org/docs/challenge-types/#http-01-challenge) for more information about challenges.

### Server Ports

if the PORT parameter is specified in the `.env` file, the server will only run HTTPS without a certificate.  See TLS section below for how to run on HTTP and HTTPS

#### TLS

Use [mkcert](https://github.com/FiloSottile/mkcert) to configure a development machine to accept local certificates.
```bash
go get github.com/FiloSottile/mkcert
mkcert -install
```
Generate certificates for localhost at 127.0.0.1
```bash
mkcert 127.0.0.1
```
Then, replace the [resources/tls-cert.pem](resources/tls-cert.pem) and [resources/tls-key.pem](resources/tls-key.pem) files with the certificates.  Update the `.env` file with the parateters below. Make sure to remove the `PORT` variable, if present.
```
HTTP_PORT=8001
HTTPS_PORT=8000
```

### Server Ports

By default, the server will run on ports 80 and 443 for http and https traffic.  All http traffic is redirected to HTTPS.

If the server handles HTTPS by providing its own certificate, use the PORT variable to specify the HTTPS port. When POST is defined, no HTTP server will be started from HTTP_PORT and certificates are not read.

##### Serve on Default TCP HTTP Ports

Run `make serve-tcp` to run on port 80 for HTTP and port 443 for HTTPS (default TCP ports).  Using these ports requires `sudo` (root) access.

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
1. Run `docker-compose up --build` to launch the application, rebuilding the parts of it that are stale.
1. Access application by opening <https://localhost:8000>.  TLS certificates will be copied to Docker.  Environment variables are used from the `.env` file.
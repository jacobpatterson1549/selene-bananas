# selene-bananas

[![Docker Image CI](https://github.com/jacobpatterson1549/selene-bananas/actions/workflows/docker-image.yml/badge.svg)](https://github.com/jacobpatterson1549/selene-bananas/actions/workflows/docker-image.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jacobpatterson1549/selene-bananas)](https://goreportcard.com/report/github.com/jacobpatterson1549/selene-bananas)
[![GoDoc](https://godoc.org/github.com/jacobpatterson1549/selene-bananas?status.svg)](https://godoc.org/github.com/jacobpatterson1549/selene-bananas)


## A Banagrams clone

A tile-based word-forming game based on the popular Banagrams game.  <https://bananagrams.com/games/bananagrams>

With WebSockets, users can play a word game together over a network.

Uses WebAssembly to manage browser logic.

A small database can be used to store user account names, encrypted passwords, and points.  Users get ten points for each game won and one point when they are beaten by someone else.  Users must use either a Postgres on Mongo database.

## Screenshot
![selene-bananas screenshot](screenshot.png)

## Dependencies

New dependencies are automatically added to [go.mod](go.mod) when the project is built.
* [pq](https://github.com/lib/pq) provides the Postgres database driver for storing user passwords and points
* [mongo-driver](https://github.com/mongodb/mongo-go-driver) provides the Mongodb database driver for storing user passwords and points
* [Gorilla WebSocket](https://github.com/gorilla/websocket) are used for bidirectional communication between users and the server
* [jwt](https://github.com/golang-jwt/jwt) is used for stateless web sessions
* [crypto](https://github.com/golang/crypto) is used to  encrypt passwords with bcrypt
* [Font-Awesome](https://github.com/FortAwesome/Font-Awesome) provides the "copyright", "github," "linkedin", and "gavel" icons on the about page; they were copied from version [5.13.0](https://github.com/FortAwesome/Font-Awesome/releases/tag/5.13.0) to [resources/template/fa](resources/template/fa).

## Build

### Running the application

* Run `make serve` to build and run the application.

* Run `make serve-tcp` to build and run on port 80 for HTTP and port 443 for HTTPS (default TCP ports).  Using these ports requires `sudo` (root) access.

### Build dependencies

[Go 1.18](https://golang.org/dl/) is used to build the application.

[Make](https://www.gnu.org/software/make/) is used to by [Makefile](Makefile) to build and runs the application. Run `make` without any arguments to build the server with the client and other resources embedded in it.  This will likely need to be done before using an IDE in order to generate some files and populate the embedded filesystem used by the the server.

[Aspell](https://github.com/GNUAspell/aspell) is used to generate the en_US dictionary to validate words on player boards.
 * Note: An integration test depends on aspell-en 2020.12.07-0.  This version is used by Docker.  Follow the steps below to install the version on your computer:
   1. Download https://ftp.gnu.org/gnu/aspell/dict/en/aspell6-en-2020.12.07-0.tar.bz2
   1. unzip the archive with `tar -xf aspell6-en-2020.12.07-0.tar.bz2`
   1. configure and install it:
   ```
   cd aspell6-en-2020.12.07-0
   ./configure
   make
   sudo make install
   ```

[Node](https://github.com/nodejs) is needed to run WebAssembly tests.

### Docker

Launching the application with [Docker](https://www.docker.com) requires minimal configuration.

1. Install [docker-compose](https://github.com/docker/compose)
1. Set environment variables in the `.env` file in project root (next to Dockerfile).
    ```
    PORT=8000
    NO_TLS_REDIRECT=true
    HTTP_PORT=8001
    HTTPS_PORT=8000

    DATABASE_URL=postgres://selene:selene123@127.0.0.1:5432/selene_bananas_db
    POSTGRES_DB=selene_bananas_db
    POSTGRES_USER=selene
    POSTGRES_PASSWORD=selene123
    POSTGRES_PORT=54320
    ```

Optionally, start database first:
1. Run `docker-compose up postgres-db` to launch a Postgres database in docker.  To run a mongo database, run `docker-compose up mongo-db` after changing the `DATABASE_URL` to `mongodb://selene-bananas-db-mongo:27017/` in `docker-compose.yml` file.

1. Run `docker-compose up --build web` to launch the application, rebuilding parts of it that are stale.
1. Access application by opening <http://127.0.0.1:8000>.  TLS certificates will be copied to Docker.  Environment variables are used from the `.env` file.
1. Run `docker-compose down` after running the application to ensure the web database servers shut down.

To run a Mongo database instead of Postgres, make the following changes:
* In `docker-compose.yml`: change the line after `depends-on:` from `postgres-db:` to `mongo-db:`
* In `.env`: set `DATABASE_URL` to `mongodb://127.0.0.1:27017/`

### Environment Configuration

Environment variables in the `.env` file are needed to customize the server.

Minimal config:
```
PORT=8000
NO_TLS_REDIRECT=true
```

For development, set `CACHE_SECONDS` to `0` to not cache static and template resources.

#### Database

Optionally, the app stores user information in either a a Postgresql or Mongodb database.  The database to use is specified by the `DATABASE_URL` environment argument.  When the app starts, the database is initialized.  For SQL databases, files in the [resources/sql](resources/sql) folder are run to ensure database objects functions are fresh.

##### localhost

# Mongodb

A Mongo database is very easy to set up when using the docker image.
* Run `docker-compose up --build mongo-db` to start the database in a terminal.
* Run the following command to query the database in a terminal: `docker exec -it selene-bananas-db-mongo mongo`
* Set the `DATABASE_URL` environment variable to `mongodb://127.0.0.1:27017/` before starting the web server to connect to the local mongo server.

# Postgres

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

#### HTTPS

The app can be run on HTTP over TLS (HTTPS). If running on TLS, most HTTP requests are redirected to HTTPS.

If the server handles HTTPS by providing its own certificate, use the PORT variable to specify the HTTPS port. When POST is defined, no HTTP server will be started from HTTP_PORT and certificates are not read.  Use this in combination weth `NO_TLS_REDIRECT=true` to prevent the server trying to check the TLS headers on requests.

##### Mkcert

Use [mkcert](https://github.com/FiloSottile/mkcert) to configure a development machine to accept local certificates.
```bash
go get github.com/FiloSottile/mkcert
mkcert -install
```
Generate certificates for localhost at 127.0.0.1
```bash
mkcert 127.0.0.1
```
Then, replace the `resources/tls-cert.pem` and `resources/tls-key.pem` files with the certificates.  Update the `.env` file with the parameters below. Make sure to remove the `PORT` variable, if present.
```
HTTP_PORT=8001
HTTPS_PORT=8000
```

##### ACME

The server can verify its identity over HTTP to pass a Automatic Certificate Management Environment (ACME) HTTP-01 challenge.  **Using a local certificate generated in the previous step by mkcert**, add the ACME environment parameters listed below with necessary values to the `.env` file.  This makes the HTTP server respond to challenge requests correctly.  After the certificates are created, remove the ACME_* parameter and replace the `resources/tls-cert.pem` and `resources/tls-key.pem` files with the certificates. See [letsencrypt.org](https://letsencrypt.org/docs/challenge-types/#http-01-challenge) for more information about challenges.
```
ACME_CHALLENGE_TOKEN=token123
ACME_CHALLENGE_KEY=s3cr3t_key
```

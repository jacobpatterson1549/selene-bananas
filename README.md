# ![selene-bananas favicon](static/favicon.ico) selene-bananas

[![Build Status](https://travis-ci.org/jacobpatterson1549/selene-bananas.svg?branch=master)](https://travis-ci.org/jacobpatterson1549/selene-bananas)
[![Go Report Card](https://goreportcard.com/badge/github.com/jacobpatterson1549/selene-bananas)](https://goreportcard.com/report/github.com/jacobpatterson1549/selene-bananas)
[![GoDoc](https://godoc.org/github.com/jacobpatterson1549/selene-bananas?status.svg)](https://godoc.org/github.com/jacobpatterson1549/selene-bananas)


## A Banagrams clone

A tile-based word-forming game based on the popular Banagrams game.  https://bananagrams.com/games/bananagrams

With WebSockets, users can play a word game together over a network.

## Dependencies

New dependencies are automatically added to [go.mod](go.mod) when the project is built.
* [pq](https://github.com/lib/pq) provides the Postgres driver for storing user passwords and points
* [Gorilla Websockets](https://github.com/gorilla/websocket) are used for bidirectional communication between users and the server
* [jwt-go](https://github.com/dgrijalva/jwt-go) is used for stateless web sessions
* [crypto](https://github.com/golang/crypto) is used to  encrypt passwords with the Bcrypt one-way function
* [Font-Awesome](https://github.com/FortAwesome/Font-Awesome) provides the "copyright", "github," and, "linkedin" icons on the about page; they were copied from version [5.13.0](https://github.com/FortAwesome/Font-Awesome/releases/tag/5.13.0) to [static/fa](static/fa).

## Build/Run

### Environment configuration

Environment variables are needed to customize the server.  Sample config:
```
APPLICATION_NAME=selene_bananas
DATABASE_URL=postgres://selene:selene123@127.0.0.1:54320/selene_bananas_db?sslmode=disable
PORT=8000 # Server web port
```

It is recommended to install the [wamerican-small](https://packages.debian.org/buster/wamerican-small) package.  This package provides /usr/share/dict/american-english-small, which is the default location of the word list.  Lowercase words are read from the word list for checking valid words in the game.  This can be overridden by providing the `WORDS_FILE` environment variable.

### Database

The app stores user information in a postgresql database.  Every time the app starts, files in the [sql](sql) folder are ran to ensure the table and stored functions are fresh.

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

### Make

The [Makefile](Makefile) runs the application locally.  This requires Go and a Postgres database to be installed.  Run `make serve` to build and run the application.

### Docker

Launching the application with [Docker](https://www.docker.com) requires minimal configuration.

1. Install [docker-compose](https://github.com/docker/compose)
1. Set environment variables in a `.env` file in project root (next to Dockerfile).  Note that the DATABASE_URL will likely need the `sslmode=disable` query parameter.  Sample:
```
POSTGRES_DB=selene_bananas_db
POSTGRES_USER=selene
POSTGRES_PASSWORD=selene123
POSTGRES_PORT=54320
DATABASE_URL=postgres://<...>?sslmode=disable
```
3. Run `docker-compose up` to launch the application.
1. Access application by opening <http://localhost:8000>.

### Heroku

Heroku is a platform-as-a-service tool that can be used to run the server on the Internet.

Steps to run on Heroku:

1. Provision a new app on [Heroku](https://dashboard.heroku.com/apps).  The name of the application is referenced as HEROKU_APP_NAME in the steps below
1. Provision a [Heroku Postgres](https://www.heroku.com/postgres) **add-on** on the **Overview** (main) tab for the app.
1. Configure additional environment variables, such as APPLICATION_NAME on the **Settings** tab.  The PORT and DATABASE_URL variables are automatically configured, although the PORT variable is not displayed.
1. Connect the app to this GitHub repository on the **Deploy** tab.  Use the GIT_URL, https://github.com/jacobpatterson1549/selene-bananas.git.
1. In a terminal, with [heroku-cli](https://devcenter.heroku.com/articles/heroku-cli) run the command below.  This builds the and deploys the code to Heroku using a docker image.
```
git clone GIT_URL
heroku git:remote -a HEROKU_APP_NAME
heroku stack:set container
git push heroku master
```
# ![selene-bananas favicon](static/favicon.ico) selene-bananas

[![Build Status](https://travis-ci.org/jacobpatterson1549/selene-bananas.svg?branch=master)](https://travis-ci.org/jacobpatterson1549/selene-bananas)
[![Go Report Card](https://goreportcard.com/badge/github.com/jacobpatterson1549/selene-bananas)](https://goreportcard.com/report/github.com/jacobpatterson1549/selene-bananas)
[![GoDoc](https://godoc.org/github.com/jacobpatterson1549/selene-bananas?status.svg)](https://godoc.org/github.com/jacobpatterson1549/selene-bananas)


## A Banagrams clone
https://bananagrams.com/games/bananagrams
Uses WebSockets to allow multiple users to play a word game over a network.

## Dependencies
New dependencies are automatically added to [go.mod](go.mod) when the project is built.
* [pq](https://github.com/lib/pq) (PostgreSQL Driver)
* [bcrypt](https://github.com/golang/crypto) (password encryption)
* [Gorilla WebSocket](https://github.com/gorilla/websocket) (game websocket)
* [Font-Awesome](https://github.com/FortAwesome/Font-Awesome) (icons on about page)

## Build/Run

### Environment configuration
Environment properties are needed to customize server characteristics.  Sample config:
```
APPLICATION_NAME=selene_bananas
DATABASE_URL=postgres://selene:selene123@127.0.0.1:54320/selene_bananas_db?sslmode=disable
PORT=8000 # Server port
```

### Make
Run `make` to build and run the application.  Requires Go to be installed and a Postgres database to be installed.

### Docker
Launching the application with [Docker](https://www.docker.com) requires minimal configuration. 
1. Install [docker-compose](https://github.com/docker/compose)
1. Set environment variables in a `.env` file in project root (next to Dockerfile). Sample:
```
POSTGRES_DB=selene_bananas_db
POSTGRES_USER=selene
POSTGRES_PASSWORD=selene123
POSTGRES_PORT=54320
```
3. Run `docker-compose up` to launch the application.
1. Access application by opening <http://localhost:8000>.

### Heroku
1. Provision a new app on [Heroku](https://dashboard.heroku.com/apps).
1. Provision a [Heroku Postgres](https://www.heroku.com/postgres) **add-on** on the **Overview** (main) tab for the app.
1. Configure additional environment variables on the **Settings** tab.  The PORT and DATABASE_URL variable is automatically configured.
1. Connect the app to this GitHub repository on the **Deploy** tab.
1. Trigger a **Manual deploy** on the **Deploy** tab.

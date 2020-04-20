FROM golang:1.13-buster AS build

RUN apt-get update && apt-get install -y wamerican

WORKDIR /app

# fetch dependencies first so they will not have to be refetched when other source code changes
COPY go.mod go.sum /app/

RUN go mod download

COPY . /app/

# build server without links to C libraries
RUN CGO_ENABLED=0 go build -o /app/selene_bananas main.go

FROM scratch

WORKDIR /app

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY --from=build /usr/share/dict/american-english /usr/share/dict/american-english

COPY --from=build /app /app

# use exec form to not run from shell, which scratch image does not have
CMD ["/app/selene_bananas"]
FROM golang:1.13-buster AS build

WORKDIR /app

# fetch dependencies first so they will not have to be refetched when other source code changes
COPY go.mod go.sum /app/

RUN go mod download

COPY . /app/

# build server without links to C libraries
RUN CGO_ENABLED=0 go build -o /app/selene_bananas main.go

FROM scratch

WORKDIR /app

# TODO: Reorganize Dockerfile to do these first two copies in stagest before building
COPY --from=build /etc/ssl/certs/ca-certificates.crt .

COPY --from=build /usr/share/dict/american-english .

COPY --from=build /app /app

# use exec form to not run from shell, which scratch image does not have
CMD ["/app/selene_bananas"]
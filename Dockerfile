FROM golang:1.13-alpine AS build

WORKDIR /app

# fetch dependencies first so they will not have to be refetched when other source code changes
COPY go.mod go.sum /app/

RUN go mod download

COPY . /app/

# build server without links to C libraries
RUN CGO_ENABLED=0 go build -o /app/selene_bananas go/cmd/server/main.go

FROM scratch

WORKDIR /app

# copy the x509 certificate file for Alpine Linux to allow server to make https requests
COPY --from=build /etc/ssl/cert.pem /etc/ssl/cert.pem

COPY --from=build /app /app

# use exec form to not run from shell, which scratch image does not have
CMD ["/app/selene_bananas"]
FROM golang:1.13-buster

RUN apt-get update && apt-get install -y wamerican

WORKDIR /app

COPY go.mod go.sum /app/

RUN go mod download

COPY . /app/

RUN CGO_ENABLED=0 go build -o /app/selene_bananas main.go

FROM scratch

WORKDIR /app

COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY --from=0 /usr/share/dict/american-english /usr/share/dict/american-english

COPY --from=0 /app /app

CMD ["/app/selene_bananas"]
FROM golang:1.13-buster

RUN apt-get update && apt-get install -y wamerican-small=2018.04.16-1

WORKDIR /app

COPY go.mod go.sum /app/

RUN go mod download

COPY . /app

RUN go test ./...

RUN CGO_ENABLED=0 go build -o /app/selene_bananas go/main.go

FROM alpine:3.11

WORKDIR /app

COPY --from=0 /usr/share/dict/american-english-small words

COPY --from=0 /app .

CMD ["/app/selene_bananas", "-words-file=/app/words"]
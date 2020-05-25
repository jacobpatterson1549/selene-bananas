FROM golang:1.13-buster

RUN apt-get update && \
    apt-get install -y wamerican-small=2018.04.16-1

WORKDIR /app

COPY go.mod go.sum /app/

RUN go mod download

COPY go /app/go

RUN go test ./...

RUN CGO_ENABLED=0 go build -o /app/selene-bananas go/*.go

FROM alpine:3.11

WORKDIR /app

COPY --from=0 /app/selene-bananas /usr/share/dict/american-english-small  /app/

COPY . /app/
# COPY sql static html js /app/ # TODO: only copy these folders as folders
# RUN ls /app -l

CMD /app/selene-bananas -words-file=/app/american-english-small
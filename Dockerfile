FROM golang:1.13-buster

RUN apt-get update && \
    apt-get install -y \
        wamerican-small=2018.04.16-1

WORKDIR /app

COPY go.mod go.sum /app/

RUN go mod download

COPY go /app/go

RUN GOOS=js \
    GOARCH=wasm \
        go build \
            -o /app/main.wasm \
            /app/go/cmd/ui/main.go; \
    CGO_ENABLED=0 \
        go build \
            -o /app/main \
            /app/go/cmd/server/*.go;

FROM alpine:3.11

WORKDIR /app

COPY --from=0 \
    /app/main \
    /app/main.wasm \
    /usr/local/go/misc/wasm/wasm_exec.js \
    /usr/share/dict/american-english-small \
    /app/

COPY . /app/
# COPY sql static html js /app/ # TODO: only copy these folders as folders, while excluding go/*, go.mod, go.sum
# RUN ls /app -lh

CMD /app/main \
        -words-file /app/american-english-small
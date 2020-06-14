FROM golang:1.14-buster

RUN apt-get update; \
    apt-get install -y \
        nodejs \
        wamerican-small=2018.04.16-1;

WORKDIR /app

COPY go/go.mod go/go.sum /app/

RUN go mod download 

COPY go /app

RUN GOOS=js GOARCH=wasm \
        go test -exec=/usr/local/go/misc/wasm/go_js_wasm_exec \
			github.com/jacobpatterson1549/selene-bananas/go/ui/... --cover; \
    go test ./... --cover; \
    GOOS=js GOARCH=wasm \
        go build \
            -o /app/main.wasm \
            /app/cmd/ui/*.go; \
    CGO_ENABLED=0 \ 
        go build \
            -o /app/main \
            /app/cmd/server/*.go;

FROM alpine:3.11

WORKDIR /app

COPY --from=0 \
    /app/main \
    /app/main.wasm \
    /usr/local/go/misc/wasm/wasm_exec.js \
    /usr/share/dict/american-english-small \
    /app/

COPY . /app/
# COPY sql static html /app/ # TODO: only copy these folders as folders, while excluding go/*

CMD /app/main \
        -words-file /app/american-english-small
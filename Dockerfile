# download golang dependencies, add node & bash to run wasm tests and american-english word list
FROM golang:1.14-alpine3.12 \
    AS BUILDER
WORKDIR /app
COPY \
    go.mod \
    go.sum \
    /app/
RUN go mod download && \
    apk add \
        nodejs bash \
    # TODO: use package from main repo, not edge:testing
    && apk add -X http://dl-cdn.alpinelinux.org/alpine/edge/testing words-en 
        # words-en

# create version, run tests, and build the applications
COPY . /app
RUN tar -cf - . | md5sum | cut -c -32 > /app/version && \
    echo version $(cat /app/version) && \
    GOOS=js GOARCH=wasm \
        go test -exec=/usr/local/go/misc/wasm/go_js_wasm_exec \
            $(GOOS=js GOARCH=wasm go list ./... | grep ui)/... --cover && \
    CGO_ENABLED=0 \
        go test ./... --cover && \
    GOOS=js GOARCH=wasm \
        go build \
            -o /app/main.wasm \
            /app/cmd/ui/*.go && \
    CGO_ENABLED=0 \
        go build \
            -o /app/main \
            /app/cmd/server/*.go

# copy necessary files and folders to a minimal build image
FROM scratch
WORKDIR /app
COPY --from=BUILDER \
    /app/main \
    /app/main.wasm \
    /app/version \
    /usr/local/go/misc/wasm/wasm_exec.js \
    /usr/share/dict/american-english \
    /app/
COPY --from=BUILDER \
    /app/resources \
    /app/resources/
ENTRYPOINT [ \
    "/app/main", \
    "-words-file=/app/american-english", \
    "-version-file=/app/version" ]

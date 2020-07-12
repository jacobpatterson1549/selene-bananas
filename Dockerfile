# download golang dependencies, add node to run wasm tests and wamerican-large word list
FROM golang:1.14-buster \
    AS BUILDER
WORKDIR /app
COPY \
    go.mod \
    go.sum \
    /app/
RUN go mod download && \
    apt-get update && \
    apt-get install \
        --no-install-recommends \
        -y \
            nodejs \
            wamerican-large=2018.04.16-1

# create version, run tests, and build the applications
COPY . /app
RUN tar -cf - . | md5sum | cut -c -32 > /app/version && \
    echo version $(cat /app/version) && \
    GOOS=js GOARCH=wasm \
        go test -exec=/usr/local/go/misc/wasm/go_js_wasm_exec \
			github.com/jacobpatterson1549/selene-bananas/ui/... --cover && \
    CGO_ENABLED=0 \
        go test ./... --cover && \
    CGO_ENABLED=0 \
        go test ./... -bench=. && \
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
    /usr/share/dict/american-english-large \
    /app/
COPY --from=BUILDER \
    /app/resources \
    /app/resources/
ENTRYPOINT [ \
    "/app/main", \
    "-words-file=/app/american-english-large", \
    "-version-file=/app/version" ]

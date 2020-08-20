# download golang dependencies, add node to run wasm tests, american-english word list, and tinygo to build a small ui binary
FROM golang:1.14-buster \
    AS BUILDER
SHELL ["/bin/bash", "-eo", "pipefail", "-c"]
WORKDIR /app
COPY \
    go.mod \
    go.sum \
    /app/
RUN go mod download \
    && apt-get update \
    && apt-get install \
        --no-install-recommends \
        -y \
            nodejs=10.21.0~dfsg-1~deb10u1 \
            wamerican-large=2018.04.16-1 \
    && wget -q https://github.com/tinygo-org/tinygo/releases/download/v0.14.0/tinygo_0.14.0_amd64.deb \
    && dpkg -i tinygo_0.14.0_amd64.deb

# create version, run tests, and build the applications
COPY \
    . \
    /app/
RUN tar -c . | md5sum | cut -c -32 \
        | tee /app/version \
        | xargs echo version \
    && GOOS=js GOARCH=wasm \
            go list ./... | grep ui \
        | GOOS=js GOARCH=wasm \
            xargs go test --cover \
                -exec=/usr/local/go/misc/wasm/go_js_wasm_exec \
    && go list ./... | grep -v ui \
        | CGO_ENABLED=0 \
            xargs go test --cover \
    && GOOS=js GOARCH=wasm \
            go list ./... | grep cmd/ui \
        | GOOS=js GOARCH=wasm \
            xargs tinygo build \
                -o /app/main.wasm \
    && go list ./... | grep cmd/server \
        | CGO_ENABLED=0 \
            xargs go build \
                -o /app/main

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

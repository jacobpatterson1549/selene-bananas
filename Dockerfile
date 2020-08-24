# download golang dependencies, add node to run wasm tests, american-english word list
FROM golang:1.14-buster \
    AS BUILDER
SHELL ["/bin/bash", "-eo", "pipefail", "-c"]
WORKDIR /app
COPY \
    go.mod \
    go.sum \
    ./
RUN go mod download \
    && apt-get update \
    && apt-get install \
        --no-install-recommends \
        -y \
            nodejs=10.21.0~dfsg-1~deb10u1 \
            wamerican-large=2018.04.16-1

# create version, run tests, build the applications, and link resources to build folder
COPY \
    . \
    ./
RUN mkdir build \
    && touch build/version \
    && tar -c . | md5sum | cut -c -32 \
        | tee build/version \
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
            xargs go build \
                -o build/main.wasm \
    && go list ./... | grep cmd/server \
        | CGO_ENABLED=0 \
            xargs go build \
                -o build/main \
    && mv resources build

# copy files to a minimal build image
FROM scratch
WORKDIR /app
COPY --from=BUILDER \
    /app/build \
    /usr/local/go/misc/wasm/wasm_exec.js \
    /usr/share/dict/american-english-large \
    /app/
ENTRYPOINT [ \
    "/app/main", \
    "-words-file=/app/american-english-large", \
    "-version-file=/app/version" ]

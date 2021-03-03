# download golang dependencies, add node to run wasm tests, american-english-large word list
FROM golang:1.16-buster \
    AS BUILDER
WORKDIR /app
COPY \
    go.mod \
    go.sum \
    ./
RUN apt-get update && apt-get install -y \
        --no-install-recommends \
        nodejs=10.24.0~dfsg-1~deb10u1 \
        wamerican-large=2018.04.16-1 \
    && rm -rf /var/lib/apt/lists/* \
    && go mod download

# build the server with embedded resources
COPY \
    . \
    ./
RUN make build/main \
    GO_ARGS="CGO_ENABLED=0"

# copy files to a minimal build image
FROM scratch
WORKDIR /app
COPY --from=BUILDER \
    /app/build/main \
    ./
ENTRYPOINT [ "/app/main" ]

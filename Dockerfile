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
            nodejs=10.23.1~dfsg-1~deb10u1 \
            wamerican-large=2018.04.16-1

# build the application without static libraries (and create version hash, test, copy resources instead of linking)
COPY \
    . \
    ./
RUN make build \
    GO_ARGS="CGO_ENABLED=0" \
    LINK="cp -R" \
    -j 2

# copy files to a minimal build image
FROM scratch
WORKDIR /app
COPY --from=BUILDER \
    /app/build \
    /usr/share/dict/american-english-large \
    /app/
ENTRYPOINT [ \
    "/app/main", \
    "-words-file=/app/american-english-large", \
    "-version-file=/app/version" ]

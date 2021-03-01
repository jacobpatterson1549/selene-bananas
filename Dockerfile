# download golang dependencies, add node to run wasm tests, american-english-large word list
FROM golang:1.16-buster \
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
            nodejs=10.24.0~dfsg-1~deb10u1 \
            wamerican-large=2018.04.16-1

# build the application without static libraries
COPY \
    . \
    ./
RUN make \
    GO_ARGS="CGO_ENABLED=0"

# copy files to a minimal build image
FROM scratch
WORKDIR /app
COPY --from=BUILDER \
    /app/build/main \
    ./
ENTRYPOINT [ "/app/main" ]

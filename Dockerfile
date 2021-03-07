# download dependencies:
# make and bash to run the Makefile
# mailcap to populate http mime types
# nodejs to run client wasm tests
# aspell and aspell-en for game word list
# download go dependencies for source code
FROM golang:1.16-alpine3.13 \
    AS BUILDER
WORKDIR /app
COPY \
    go.mod \
    go.sum \
    ./
RUN apk add --no-cache \
        make=4.3-r0 \
        bash=5.1.0-r0 \
        mailcap=2.1.49-r0 \
        nodejs=14.16.0-r0 \
        aspell=0.60.8-r0 \
        aspell-en=2020.12.07-r0 \
    && go mod download

# build the server, delete build cache
COPY \
    . \
    ./
RUN make build/main \
        GO_ARGS="CGO_ENABLED=0" \
    && go clean -cache

# copy the server to a minimal build image
FROM scratch
WORKDIR /app
COPY --from=BUILDER \
    /app/build/main \
    ./
ENTRYPOINT [ "/app/main" ]

# download dependencies:
# make and bash to run the Makefile
# nodejs to run client wasm tests
# aspell and aspell-en for game word list
# download go dependencies for source code
FROM golang:1.27-alpine3.24 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN apk add --no-cache \
        make=~4.4.1-r4 \
        bash=~5.3.9-r1 \
        nodejs=~24.18.1-r0 \
        aspell=~0.60.8.2-r0 \
        aspell-en=2026.02.25-r0 \
    && go mod download

# build the server, delete build cache
COPY . ./
RUN make build/selene-bananas \
        GO_ARGS="CGO_ENABLED=0" \
    && go clean -cache

# copy the server to a minimal build image
FROM scratch
WORKDIR /app
COPY --from=builder /etc/ssl/cert.pem /etc/ssl/cert.pem
COPY --from=builder app/build/selene-bananas ./
ENTRYPOINT [ "/app/selene-bananas" ]

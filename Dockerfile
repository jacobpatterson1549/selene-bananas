# download dependencies:
# make and bash to run the Makefile
# nodejs to run client wasm tests
# aspell and aspell-en for game word list
# download go dependencies for source code
FROM golang:1.24-alpine3.21 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN apk add --no-cache \
        make=~4.4.1-r2 \
        bash=~5.2.37-r0 \
        nodejs=~22.15.1-r0 \
        aspell=~0.60.8.1-r0 \
        aspell-en=2020.12.07-r0 \
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

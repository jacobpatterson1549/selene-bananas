# download dependencies:
# make and bash to run the Makefile
# nodejs to run client wasm tests
# aspell and aspell-en for game word list
# download go dependencies for source code
FROM golang:1.18-alpine3.15 AS BUILDER
WORKDIR /app
COPY go.mod go.sum ./
RUN apk add --no-cache \
        make=~4.3 \
        bash=~5.1 \
        nodejs=~16 \
        aspell=~0.60 \
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
COPY --from=BUILDER /etc/ssl/cert.pem /etc/ssl/cert.pem
COPY --from=BUILDER app/build/selene-bananas ./
ENTRYPOINT [ "/app/selene-bananas" ]

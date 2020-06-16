# initialize environment, add node to run wasm tests and wamerican-large word list, download golang dependencies
FROM golang:1.14-buster
RUN apt-get update && \
    apt-get install \
        --no-install-recommends \
        -y \
            nodejs \
            wamerican-large=2018.04.16-1
WORKDIR /app
COPY go/go.mod go/go.sum /app/
RUN go mod download 

# run tests and build the applications
COPY go /app
RUN GOOS=js GOARCH=wasm \
        go test -exec=/usr/local/go/misc/wasm/go_js_wasm_exec \
			github.com/jacobpatterson1549/selene-bananas/go/ui/... --cover && \
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
COPY --from=0 \
    /app/main \
    /app/main.wasm \
    /usr/local/go/misc/wasm/wasm_exec.js \
    /usr/share/dict/american-english-large \
    /
COPY sql  /sql/
COPY static /static/
COPY html /html/

# run the server
CMD ["/main", "-words-file", "/american-english-large"]
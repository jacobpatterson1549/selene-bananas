.PHONY: all test install serve clean

all: install

test:
	go test ./... -v

wasm:
	ln -fs \
		$(shell go env GOROOT)/misc/wasm/wasm_exec.js \
		wasm_exec.js
	GOOS=js \
	GOARCH=wasm \
	go build \
		-o main.wasm \
		go/cmd/ui/*.go

install: test wasm
	go build \
		-o main \
		go/cmd/server/*.go

serve: install
	export $(shell grep -v '^#' .env | xargs) && \
	./main

clean:
	rm -f \
		go/cmd/server/__debug_bin \
		wasm_exec.js \
		main.wasm \
		main
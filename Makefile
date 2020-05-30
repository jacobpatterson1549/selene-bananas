.PHONY: all test gopherjs install serve clean

all: install

test:
	GOOS=js \
	GOARCH=wasm go \
		test github.com/jacobpatterson1549/selene-bananas/go/ui/...
	go test ./... --cover

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
		main.wasm \
		wasm_exec.js \
		main
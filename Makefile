.PHONY: all test-wasm test wasm install serve clean

all: install

# requires node
test-wasm:
	cd go; \
		GOOS=js \
		GOARCH=wasm \
		go test -exec=$(shell go env GOROOT)/misc/wasm/go_js_wasm_exec \
			github.com/jacobpatterson1549/selene-bananas/go/ui/... --cover

test: #test-wasm
	cd go; \
	go test ./... --cover

wasm:
	ln -fs \
		$(shell go env GOROOT)/misc/wasm/wasm_exec.js \
		wasm_exec.js
	cd go; \
		GOOS=js \
		GOARCH=wasm \
		go build \
			-o ../main.wasm \
			cmd/ui/*.go

install: test wasm
	cd go; \
		go build \
		-o ../main \
		cmd/server/*.go

serve: install
	export $(shell grep -v '^#' .env | xargs) && \
		./main

clean:
	rm -f \
		go/cmd/server/__debug_bin \
		main.wasm \
		wasm_exec.js \
		main
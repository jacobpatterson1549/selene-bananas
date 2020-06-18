.PHONY: all test-wasm test install serve clean

all: install

CWD := $(shell pwd)
GO_DIR := $(CWD)/go
GOROOT := $(shell go env GOROOT)
WASM := GOOS=js GOARCH=wasm

test-wasm:
	cd $(GO_DIR); \
	$(WASM) go test -exec=$(GOROOT)/misc/wasm/go_js_wasm_exec \
		$(shell $(WASM) go list ./...  | grep go/ui)/... --cover

test:
	cd $(GO_DIR); \
	go test $(shell go list ./...) --cover

wasm_exec.js:
	ln -fs \
		$(GOROOT)/misc/wasm/wasm_exec.js \
		wasm_exec.js

main.wasm: test-wasm
	cd $(GO_DIR); \
	$(WASM) go build \
			-o $(CWD)/main.wasm \
			cmd/ui/*.go

main: test
	cd $(GO_DIR); \
	go build \
		-o $(CWD)/main \
		cmd/server/*.go

install: main main.wasm wasm_exec.js

serve: install
	export $(shell grep -v '^#' .env | xargs) && \
		./main

clean:
	rm -f \
		go/cmd/server/__debug_bin \
		main.wasm \
		wasm_exec.js \
		main
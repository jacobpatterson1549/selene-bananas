.PHONY: all test-wasm test install serve clean

all: install

CWD := $(shell pwd)
GO_DIR := $(CWD)/go
GOROOT := $(shell go env GOROOT)
WASM := GOOS=js GOARCH=wasm
WASM_PKGS := $(shell $(WASM) go list ./...) # TODO: Not used
SERVER_PKGS := $(shell go list ./...)

# requires node
test-wasm:
	cd $(GO_DIR); \
	$(WASM) go test -exec=$(GOROOT)/misc/wasm/go_js_wasm_exec \
		github.com/jacobpatterson1549/selene-bananas/go/ui/... --cover

test:
	cd $(GO_DIR); \
	go test $(SERVER_PKGS) --cover

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
.PHONY: all test-wasm test bench build serve serve-tcp clean

GO_LIST := go list
GO_TEST := go test --cover # -race
GO_BUILD := go build # -race
GO_BENCH := go test -bench=.
GO_WASM_ARGS := GOOS=js GOARCH=wasm
GO_WASM_PATH := $(shell go env GOROOT)/misc/wasm

all: build

test-wasm:
	$(GO_WASM_ARGS) $(GO_LIST) ./... | grep ui \
		| $(GO_WASM_ARGS) xargs $(GO_TEST) \
			-exec=$(GO_WASM_PATH)/go_js_wasm_exec

test:
	$(GO_LIST) ./... \
		| xargs $(GO_TEST)

bench:
	$(GO_BENCH) ./...

wasm_exec.js:
	ln -fs \
		$(GO_WASM_PATH)/$@ \
		$@

main.wasm: test-wasm
	$(GO_WASM_ARGS) $(GO_LIST) ./... | grep cmd/ui \
		| $(GO_WASM_ARGS) xargs $(GO_BUILD) \
			-o $@

main: test
	$(GO_LIST) ./... | grep cmd/server \
		| xargs $(GO_BUILD) \
			-o $@

build: main main.wasm wasm_exec.js

serve: build
	export $(shell grep -s -v '^#' .env | xargs) \
		&& ./main

serve-tcp: build
	sudo setcap 'cap_net_bind_service=+ep' main
	export $(shell grep -s -v '^#' .env | xargs \
			| xargs -I {} echo "{} HTTP_PORT=80 HTTPS_PORT=443") \
		&& sudo -E ./main

clean:
	rm -f \
		cmd/server/__debug_bin \
		wasm_exec.js \
		main.wasm \
		main

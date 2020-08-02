.PHONY: all test-wasm test bench install serve serve-tcp clean

all: install

GOWASMPATH := $(shell go env GOROOT)/misc/wasm
GOLIST := go list
GOTEST := go test --cover # -race
GOBUILD := go build # -race
GOBENCH := go test -bench=.
GOWASMARGS := GOOS=js GOARCH=wasm

test-wasm:
	$(GOWASMARGS) $(GOLIST) ./... | grep ui \
		| $(GOWASMARGS) xargs $(GOTEST) -exec=$(GOWASMPATH)/go_js_wasm_exec

test:
	$(GOLIST) ./... \
		| xargs $(GOTEST)

bench:
	$(GOBENCH) ./...

wasm_exec.js:
	ln -fs \
		$(GOWASMPATH)/wasm_exec.js \
		wasm_exec.js

main.wasm: test-wasm
	$(GOWASMARGS) $(GOBUILD) \
			-o main.wasm \
			cmd/ui/*.go

main: test bench
	$(GOBUILD) \
		-o main \
		cmd/server/*.go

install: main main.wasm wasm_exec.js

serve: install
	export $(shell grep -s -v '^#' .env | xargs) \
		&& ./main

serve-tcp: install
	sudo setcap 'cap_net_bind_service=+ep' main
	export $(shell grep -s -v '^#' .env | xargs \
			| xargs -I {} echo "{} HTTP_PORT=80 HTTPS_PORT=443") \
		&& sudo -E ./main

clean:
	rm -f \
		cmd/server/__debug_bin \
		main.wasm \
		wasm_exec.js \
		main

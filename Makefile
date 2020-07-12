.PHONY: all test-wasm test bench install serve serve-tcp clean

all: install

GOROOT := $(shell go env GOROOT)
WASM := GOOS=js GOARCH=wasm

test-wasm:
	$(WASM) go test -exec=$(GOROOT)/misc/wasm/go_js_wasm_exec \
		$(shell $(WASM) go list ./...  | grep ui)/... --cover

test:
	go test $(shell go list ./...) --cover

banch:
	go test ./... -bench=WordChecker

wasm_exec.js:
	ln -fs \
		$(GOROOT)/misc/wasm/wasm_exec.js \
		wasm_exec.js

main.wasm: test-wasm
	$(WASM) go build \
			-o main.wasm \
			cmd/ui/*.go

main: test bench
	go build \
		-o main \
		cmd/server/*.go

install: main main.wasm wasm_exec.js

serve: install
	export $(shell grep -s -v '^#' .env | xargs) && \
		./main

serve-tcp: install
	sudo setcap 'cap_net_bind_service=+ep' main
	export $(shell \
		echo $(shell grep -s -v '^#' .env | xargs) \
		" HTTP_PORT=80 HTTPS_PORT=443" \
		) && \
		sudo -E ./main

clean:
	rm -f \
		cmd/server/__debug_bin \
		main.wasm \
		wasm_exec.js \
		main

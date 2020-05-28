.PHONY: all test gopherjs install serve clean

all: install

test:
	GOOS=js GOARCH=wasm gopherjs test github.com/jacobpatterson1549/selene-bananas/go/ui/...
	go test ./... --cover

gopherjs:
	GOPHERJS_GOROOT=$(shell go1.12.16 env GOROOT) \
	gopherjs build \
		-o main.js \
		go/cmd/ui/*.go

install: test gopherjs
	go build \
		-o main \
		go/cmd/server/*.go

serve: install
	export $(shell grep -v '^#' .env | xargs) && \
	./main

clean:
	rm -f \
		go/cmd/server/__debug_bin \
		main.js \
		main.js.map \
		main
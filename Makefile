.PHONY: all test install serve clean

GO=/usr/local/go/bin/go

all: install

test:
	$(GO) generate github.com/jacobpatterson1549/selene-bananas/go
	$(GO) test ./... -v

install: test
	$(GO) build -o main go/main.go

serve: install
	export $(shell grep -v '^#' .env | xargs) && ./main

clean:
	rm -f main go/__debug_bin
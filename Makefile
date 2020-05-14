.PHONY: all test install serve clean

all: install

test:
	go generate github.com/jacobpatterson1549/selene-bananas/go
	go test ./... -v

install: test
	go build -o main go/main.go

serve: install
	export $(shell grep -v '^#' .env | xargs) && ./main

clean:
	rm -f main go/__debug_bin
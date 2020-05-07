.PHONY: all install serve clean

all: install

install:
	go generate github.com/jacobpatterson1549/selene-bananas/go
	go test ./... -v
	go build -o main go/main.go

serve: install
	export $(shell grep -v '^#' .env | xargs) && ./main

clean:
	rm -f main go/__debug_bin
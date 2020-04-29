.PHONY: all install serve clean

all: install

install:
	go test ./... -v
	go build -o main main.go

serve: install
	export $(shell grep -v '^#' .env | xargs) && ./main

clean:
	rm -f main __debug_bin
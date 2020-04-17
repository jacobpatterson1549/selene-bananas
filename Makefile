.PHONY: all serve clean

all: serve

serve:
	go build -o main main.go
	export $(shell grep -v '^#' .env | xargs) && ./main

clean:
	rm -f main __debug_bin
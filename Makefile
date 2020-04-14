.PHONY: all wasm serve clean

all: wasm serve

wasm: go/cmd/ui/main.go
	GOOS=js GOARCH=wasm go generate go/cmd/ui/main.go
	GOOS=js GOARCH=wasm go build -o static/main.wasm go/cmd/ui/main.go

serve: wasm
	export $(shell grep -v '^#' .env | xargs) && go run go/cmd/server/main.go

clean:
	rm -f static/main.wasm
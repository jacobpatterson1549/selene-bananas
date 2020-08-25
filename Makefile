.PHONY: test-wasm test bench mkdir-$(BUILD_DIR) serve serve-tcp clean

BUILD_DIR := build
GO_LIST := go list
GO_TEST := go test --cover # -race
GO_BUILD := go build # -race
GO_BENCH := go test -bench=.
GO_WASM_ARGS := GOOS=js GOARCH=wasm
GO_ARGS :=
GO_WASM_PATH := $(shell go env GOROOT)/misc/wasm
LINK := ln -fs
OBJS := $(addprefix $(BUILD_DIR)/,main.wasm main version wasm_exec.js resources)

$(BUILD_DIR): $(OBJS)

test-wasm:
	$(GO_WASM_ARGS) $(GO_LIST) ./... | grep ui \
		| $(GO_WASM_ARGS) xargs $(GO_TEST) \
			-exec=$(GO_WASM_PATH)/go_js_wasm_exec

test:
	$(GO_LIST) ./... | grep -v ui \
		| $(GO_ARGS) xargs $(GO_TEST)

bench:
	$(GO_BENCH) ./...

mkdir-$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

$(BUILD_DIR)/wasm_exec.js: | mkdir-$(BUILD_DIR)
	$(LINK) \
		$(GO_WASM_PATH)/$(@F) \
		$@

$(BUILD_DIR)/resources: | mkdir-$(BUILD_DIR)
	$(LINK) \
		$(PWD)/$(@F) \
		$@

$(BUILD_DIR)/version: | mkdir-$(BUILD_DIR)
	find . \
			-mindepth 2 \
			-path "*/.*" -prune -o \
			-path "./$(BUILD_DIR)/*" -prune -o \
			-type f \
			-print \
		| xargs tar -c \
		| md5sum \
		| cut -c -32 \
		| tee $@ \
		| xargs echo version

$(BUILD_DIR)/main.wasm: test-wasm | mkdir-$(BUILD_DIR)
	$(GO_WASM_ARGS) $(GO_LIST) ./... | grep cmd/ui \
		| $(GO_WASM_ARGS) xargs $(GO_BUILD) \
			-o $@

$(BUILD_DIR)/main: test | mkdir-$(BUILD_DIR)
	$(GO_LIST) ./... | grep cmd/server \
		| $(GO_ARGS) xargs $(GO_BUILD) \
			-o $@

serve: $(BUILD_DIR)
	export $(shell grep -s -v '^#' .env | xargs) \
		&& cd $(BUILD_DIR) \
		&& ./main

serve-tcp: $(BUILD_DIR)
	sudo setcap 'cap_net_bind_service=+ep' $(BUILD_DIR)/main
	export $(shell grep -s -v '^#' .env | xargs \
			| xargs -I () echo "() HTTP_PORT=80 HTTPS_PORT=443") \
		&& cd $(BUILD_DIR) \
		&& sudo -E ./main

clean:
	rm -rf $(BUILD_DIR)

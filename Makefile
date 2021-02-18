.PHONY: test-wasm test bench mkdir-build serve serve-tcp clean

BUILD_DIR := build
RESOURCES_DIR := resources
GO := go
GO_PACKAGES := ./...
GO_GENERATE_SRC := game/message/type_string.go
GO_INSTALL   := $(GO) install
GO_GENERATE  := $(GO) generate
GO_LIST      := $(GO) list
GO_TEST      := $(GO) test --cover -timeout 30s # -race # -run TestFuncName
GO_BENCH     := $(GO) test -bench=.
GO_BUILD     := $(GO) build # -race
GO_WASM_ARGS := GOOS=js GOARCH=wasm
GO_ARGS :=
GO_WASM_PATH := $(shell $(GO) env GOROOT)/misc/wasm
LINK := ln -fs
CLIENT_OBJ := main.wasm
SERVER_OBJ := main
VERSION_OBJ := version
WASM_EXEC_OBJ := wasm_exec.js
OBJS := $(addprefix $(BUILD_DIR)/,$(CLIENT_OBJ) $(SERVER_OBJ) $(VERSION_OBJ) $(WASM_EXEC_OBJ) $(RESOURCES_DIR))

$(BUILD_DIR): $(OBJS)

$(GENERATE_SRC):
	$(GO_INSTALL) $(GO_PACKAGES)
	$(GO_GENERATE) $(GO_PACKAGES)

test-wasm: $(GENERATE_SRC)
	$(GO_WASM_ARGS) $(GO_LIST) $(GO_PACKAGES) | grep ui \
		| $(GO_WASM_ARGS) xargs $(GO_TEST) \
			-exec=$(GO_WASM_PATH)/go_js_wasm_exec

test: $(GENERATE_SRC)
	$(GO_LIST) $(GO_PACKAGES) | grep -v ui \
		| $(GO_ARGS) xargs $(GO_TEST)

bench:
	$(GO_BENCH) $(GO_PACKAGES)

mkdir-build: 
	mkdir -p $(BUILD_DIR)

$(BUILD_DIR)/$(WASM_EXEC_OBJ): | mkdir-build
	$(LINK) \
		$(GO_WASM_PATH)/$(@F) \
		$@

$(BUILD_DIR)/$(RESOURCES_DIR): | mkdir-build
	$(LINK) \
		$(PWD)/$(@F) \
		$@

$(BUILD_DIR)/$(VERSION_OBJ): | mkdir-build
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
		| xargs echo $(@F)

$(BUILD_DIR)/$(CLIENT_OBJ): test-wasm | mkdir-build
	$(GO_WASM_ARGS) $(GO_LIST) $(GO_PACKAGES) | grep cmd/ui \
		| $(GO_WASM_ARGS) xargs $(GO_BUILD) \
			-o $@

$(BUILD_DIR)/$(SERVER_OBJ): test | mkdir-build
	$(GO_LIST) $(GO_PACKAGES) | grep cmd/server \
		| $(GO_ARGS) xargs $(GO_BUILD) \
			-o $@

serve: $(BUILD_DIR)
	export $(shell grep -s -v '^#' .env | xargs) \
		&& cd $(BUILD_DIR) \
		&& ./$(SERVER_OBJ)

serve-tcp: $(BUILD_DIR)
	sudo setcap 'cap_net_bind_service=+ep' $(BUILD_DIR)/$(SERVER_OBJ)
	export $(shell grep -s -v '^#' .env | xargs \
			| xargs -I {} echo "{} HTTP_PORT=80 HTTPS_PORT=443") \
		&& cd $(BUILD_DIR) \
		&& sudo -E ./$(SERVER_OBJ)

clean:
	rm -rf $(BUILD_DIR)

.PHONY:  serve serve-tcp clean

BUILD_DIR := build
RESOURCES_DIR := resources
GENERATE_SRC := game/message/type_string.go
GO_PACKAGES := ./...
GO := go
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
CLIENT_OBJ       := $(BUILD_DIR)/main.wasm
SERVER_OBJ       := $(BUILD_DIR)/main
VERSION_OBJ      := $(BUILD_DIR)/version
WASM_EXEC_OBJ    := $(BUILD_DIR)/wasm_exec.js
SERVER_TEST      := $(BUILD_DIR)/server.test
CLIENT_TEST      := $(BUILD_DIR)/client.test
SERVER_BENCHMARK := $(BUILD_DIR)/server.benchmark
SERVER_SOURCE_DIRS := cmd/server/ game/ server/ db/ resources/
CLIENT_SOURCE_DIRS := cmd/ui/     game/ ui/
SERVER_SOURCE := $(shell find $(SERVER_SOURCE_DIRS))
CLIENT_SOURCE := $(shell find $(CLIENT_SOURCE_DIRS))

$(SERVER_OBJ): $(SERVER_TEST)  $(CLIENT_OBJ) $(WASM_EXEC_OBJ) $(VERSION_OBJ) $(BUILD_DIR)/$(RESOURCES_DIR) | $(BUILD_DIR)
	$(GO_LIST) $(GO_PACKAGES) | grep cmd/server \
		| $(GO_ARGS) xargs $(GO_BUILD) \
			-o $@

$(CLIENT_OBJ): $(CLIENT_TEST) | $(BUILD_DIR)
	$(GO_WASM_ARGS) $(GO_LIST) $(GO_PACKAGES) | grep cmd/ui \
		| $(GO_WASM_ARGS) xargs $(GO_BUILD) \
			-o $@

$(WASM_EXEC_OBJ): | $(BUILD_DIR)
	$(LINK) \
		$(GO_WASM_PATH)/$(@F) \
		$@

$(SERVER_TEST): $(SERVER_SOURCE) | $(BUILD_DIR)
	$(GO_LIST) $(GO_PACKAGES) | grep -v ui \
		| $(GO_ARGS) xargs $(GO_TEST)
	touch $(SERVER_TEST)

$(CLIENT_TEST): $(CLIENT_SOURCE) | $(BUILD_DIR)
	$(GO_WASM_ARGS) $(GO_LIST) $(GO_PACKAGES) | grep ui \
		| $(GO_WASM_ARGS) xargs $(GO_TEST) \
			-exec=$(GO_WASM_PATH)/go_js_wasm_exec
	touch $(CLIENT_TEST)

$(SERVER_BENCHMARK): $(SERVER_SOURCE) | $(BUILD_DIR)
	$(GO_BENCH) $(GO_PACKAGES)
	touch $(SERVER_BENCHMARK)

$(GENERATE_SRC):
	$(GO_INSTALL) $(GO_PACKAGES)
	$(GO_GENERATE) $(GO_PACKAGES)

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

$(BUILD_DIR)/$(RESOURCES_DIR): | $(BUILD_DIR)
	$(LINK) \
		$(PWD)/$(@F) \
		$@

$(VERSION_OBJ): $(SERVER_SOURCE) $(CLIENT_SOURCE) | $(BUILD_DIR)
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
	rm -rf $(BUILD_DIR) $(GENERATE_SRC)

# list rules: https://stackoverflow.com/a/7144684/1330346
# make -pn | grep -A1 "^# makefile"| grep -v "^#\|^--" | sort | uniq

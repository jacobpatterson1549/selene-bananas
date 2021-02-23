.PHONY: serve serve-tcp clean

BUILD_DIR := build
# TODO: embed all resources in server - including main.wasm (make server depend on client)
RESOURCES_DIR    := resources
SERVER_EMBED_DIR := cmd/server/embed
GENERATE_SRC := game/message/type_string.go
VERSION_OBJ  := $(SERVER_EMBED_DIR)/version.txt
GO := go
GO_PACKAGES  := ./...
GO_INSTALL   := $(GO) install
GO_GENERATE  := $(GO) generate
GO_LIST      := $(GO) list
GO_TEST      := $(GO) test --cover -timeout 30s # -race # -run TestFuncName
GO_BENCH     := $(GO) test -bench=.
GO_BUILD     := $(GO) build # -race
GO_ARGS      :=
GO_WASM_ARGS := GOOS=js GOARCH=wasm
GO_WASM_PATH := $(shell $(GO) env GOROOT)/misc/wasm
LINK := ln -fs
SERVER_OBJ       := $(BUILD_DIR)/main
CLIENT_OBJ       := $(BUILD_DIR)/main.wasm
WASM_EXEC_OBJ    := $(BUILD_DIR)/wasm_exec.js
SERVER_TEST      := $(BUILD_DIR)/server.test
CLIENT_TEST      := $(BUILD_DIR)/client.test
SERVER_BENCHMARK := $(BUILD_DIR)/server.benchmark
RESOURCES_SRC := $(shell find $(RESOURCES_DIR) -type f)
# exclude the generated source from go sources because it is created after the version, which depends on romal source
GO_SRC_FN = find $(1) $(foreach g,$(GENERATE_SRC),-path $g -prune -o) -name *.go -print
SERVER_SRC    := $(shell $(call GO_SRC_FN, cmd/server/ game/ server/ db/))
CLIENT_SRC    := $(shell $(call GO_SRC_FN, cmd/ui/     game/ ui/))

$(SERVER_OBJ): $(SERVER_TEST) $(CLIENT_OBJ) $(WASM_EXEC_OBJ) $(BUILD_DIR)/$(RESOURCES_DIR) | $(BUILD_DIR)
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

$(SERVER_TEST): $(SERVER_SRC) $(GENERATE_SRC) | $(BUILD_DIR)
	$(GO_LIST) $(GO_PACKAGES) | grep -v ui \
		| $(GO_ARGS) xargs $(GO_TEST)
	touch $(SERVER_TEST)

$(CLIENT_TEST): $(CLIENT_SRC) | $(BUILD_DIR)
	$(GO_WASM_ARGS) $(GO_LIST) $(GO_PACKAGES) | grep ui \
		| $(GO_WASM_ARGS) xargs $(GO_TEST) \
			-exec=$(GO_WASM_PATH)/go_js_wasm_exec
	touch $(CLIENT_TEST)

$(SERVER_BENCHMARK): $(SERVER_SRC) $(GENERATE_SRC) | $(BUILD_DIR)
	$(GO_LIST) $(GO_PACKAGES) | grep -v ui \
		| $(GO_ARGS) xargs $(GO_BENCH)
	touch $(SERVER_BENCHMARK)

$(BUILD_DIR)/$(RESOURCES_DIR): $(RESOURCES_SRC) | $(BUILD_DIR) # TODO: DELETEME when all resources are embedded, including main.wasm and wasm_exec.js
	$(LINK) \
		$(PWD)/$(@F) \
		$@

$(GENERATE_SRC): $(VERSION_OBJ)
	$(GO_INSTALL) $(GO_PACKAGES)
	$(GO_GENERATE) $(GO_PACKAGES)

$(VERSION_OBJ): $(SERVER_SRC) $(CLIENT_SRC) $(RESOURCES_SRC) | $(SERVER_EMBED_DIR)
	find . \
			-mindepth 2 \
			-path "*/.*" -prune -o \
			-path "./$(BUILD_DIR)/*" -prune -o \
			-path "./$(SERVER_EMBED_DIR)/*" -prune -o \
			-path $@ -prune -o \
			-type f \
			-print \
		| xargs tar -c \
		| md5sum \
		| cut -c -32 \
		| tee $@ \
		| xargs echo $@ is

$(BUILD_DIR):
	mkdir -p $@

$(SERVER_EMBED_DIR): $(RESOURCES_DIR)
	mkdir -p $@
	# creating hard links for resources
	cp -Rlf $(PWD)/$(RESOURCES_DIR)/* $(SERVER_EMBED_DIR)

serve: $(SERVER_OBJ)
	export $(shell grep -s -v '^#' .env | xargs) \
		&& ./$(SERVER_OBJ)

serve-tcp: $(SERVER_OBJ)
	sudo setcap 'cap_net_bind_service=+ep' $(SERVER_OBJ)
	export $(shell grep -s -v '^#' .env | xargs \
			| xargs -I {} echo "{} HTTP_PORT=80 HTTPS_PORT=443") \
		&& sudo -E ./$(SERVER_OBJ)

clean:
	rm -rf $(BUILD_DIR) $(GENERATE_SRC) $(SERVER_EMBED_DIR)

# list rules: https://stackoverflow.com/a/7144684/1330346
# make -pn | grep -A1 "^# makefile"| grep -v "^#\|^--" | sort | uniq

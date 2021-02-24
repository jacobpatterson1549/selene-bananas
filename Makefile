.PHONY: serve serve-tcp clean

BUILD_DIR := build
RESOURCES_DIR    := resources
SERVER_EMBED_DIR := cmd/server/embed
GENERATE_SRC := game/message/type_string.go
VERSION_OBJ   := version.txt
CLIENT_OBJ    := main.wasm
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
SERVER_OBJ       := main
SERVER_TEST      := server.test
CLIENT_TEST      := client.test
SERVER_BENCHMARK := server.benchmark
RESOURCES_SRC := $(shell find $(RESOURCES_DIR) -type f)
# exclude the generated source from go sources because it is created after the version, which depends on romal source
GO_SRC_FN = find $(1) $(foreach g,$(GENERATE_SRC),-path $g -prune -o) -name *.go -print
SERVER_SRC := $(shell $(call GO_SRC_FN, cmd/server/ game/ server/ db/))
CLIENT_SRC := $(shell $(call GO_SRC_FN, cmd/ui/     game/ ui/))
SERVE_ARGS := $(shell grep -s -v "^\#" .env)

$(BUILD_DIR)/$(SERVER_OBJ): $(BUILD_DIR)/$(CLIENT_OBJ) $(BUILD_DIR)/$(SERVER_TEST) $(BUILD_DIR)/$(VERSION_OBJ) $(RESOURCES_SRC) | $(BUILD_DIR)
	$(GO_LIST) $(GO_PACKAGES) | grep cmd/server \
		| $(GO_ARGS) xargs $(GO_BUILD) \
			-o $@

$(BUILD_DIR)/$(CLIENT_OBJ): $(BUILD_DIR)/$(CLIENT_TEST) | $(BUILD_DIR) $(SERVER_EMBED_DIR)
	$(GO_WASM_ARGS) $(GO_LIST) $(GO_PACKAGES) | grep cmd/ui \
		| $(GO_WASM_ARGS) xargs $(GO_BUILD) \
			-o $(SERVER_EMBED_DIR)/$(@F)
	touch $@


$(BUILD_DIR)/$(SERVER_TEST): $(SERVER_SRC) $(GENERATE_SRC) | $(BUILD_DIR)
	$(GO_LIST) $(GO_PACKAGES) | grep -v ui \
		| $(GO_ARGS) xargs $(GO_TEST)
	touch $@

$(BUILD_DIR)/$(CLIENT_TEST): $(CLIENT_SRC) $(GENERATE_SRC) | $(BUILD_DIR)
	$(GO_WASM_ARGS) $(GO_LIST) $(GO_PACKAGES) | grep ui \
		| $(GO_WASM_ARGS) xargs $(GO_TEST) \
			-exec=$(GO_WASM_PATH)/go_js_wasm_exec
	touch $@

$(BUILD_DIR)/$(SERVER_BENCHMARK): $(SERVER_SRC) $(GENERATE_SRC) | $(BUILD_DIR)
	$(GO_LIST) $(GO_PACKAGES) | grep -v ui \
		| $(GO_ARGS) xargs $(GO_BENCH)
	touch $@

$(GENERATE_SRC): | $(SERVER_EMBED_DIR)
	$(GO_INSTALL) $(GO_PACKAGES)
	$(GO_GENERATE) $(GO_PACKAGES)

$(BUILD_DIR)/$(VERSION_OBJ): $(SERVER_SRC) $(CLIENT_SRC) $(RESOURCES_SRC) | $(BUILD_DIR) $(SERVER_EMBED_DIR)
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
		| tee $(SERVER_EMBED_DIR)/$(@F) \
		| xargs echo $(SERVER_EMBED_DIR)/$(@F) is
	touch $@

$(BUILD_DIR):
	mkdir -p $@

$(SERVER_EMBED_DIR): $(RESOURCES_SRC)
	mkdir -p $@
	# creating hard links, not soft symbolic links because we own the resources:
	cp -Rlf $(PWD)/$(RESOURCES_DIR)/* $@
	cp -lf LICENSE $@
	# copying wasm_exec.js because linking it may require us to have write privileges on it
	cp -f $(GO_WASM_PATH)/wasm_exec.js $@
	# the client object and version are required by go install as embedded files, even though they are built later
	touch $(SERVER_EMBED_DIR)/$(CLIENT_OBJ)
	touch $(SERVER_EMBED_DIR)/$(VERSION_OBJ)

serve: $(BUILD_DIR)/$(SERVER_OBJ)
	$(SERVE_ARGS) $(BUILD_DIR)/$(SERVER_OBJ)

serve-tcp: $(BUILD_DIR)/$(SERVER_OBJ)
	sudo setcap cap_net_bind_service=+ep $(SERVER_OBJ)
	$(SERVE_ARGS) HTTP_PORT=80 HTTPS_PORT=443 $(BUILD_DIR)/$(SERVER_OBJ)

clean:
	rm -rf $(BUILD_DIR) $(SERVER_EMBED_DIR) $(GENERATE_SRC)

# list rules: https://stackoverflow.com/a/7144684/1330346
# make -pn | grep -A1 "^# makefile"| grep -v "^#\|^--" | sort | uniq

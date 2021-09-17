.PHONY: all test clean serve serve-tcp

BUILD_DIR        := build
RESOURCES_DIR    := resources
SERVER_EMBED_DIR := cmd/server/embed
STATIC_DIR       := static
TEMPLATE_DIR     := template
SQL_DIR          := sql
LICENSE_FILE      := LICENSE
TLS_CERT_FILE     := tls-cert.pem
TLS_KEY_FILE      := tls-key.pem
GO_MOD_FILES      := go.mod go.sum
DOCKERIGNORE_FILE := .dockerignore
COPY := cp -f
LINK := $(COPY) -l
GO := go
GO_PACKAGES  := ./...
GO_INSTALL   := $(GO) install
GO_GENERATE  := $(GO) generate
GO_LIST      := $(GO) list
GO_TEST      := $(GO) test -cover -timeout 30s 
GO_BUILD     := $(GO) build
GO_ARGS      :=
GO_TEST_ARGS := # -v # -test.short # -race # -run TestFuncName 
GO_WASM_ARGS := GOOS=js GOARCH=wasm
GO_WASM_PATH := $(shell $(GO) env GOROOT)/misc/wasm
NODE_EXEC    := node $(GO_WASM_PATH)/wasm_exec.js
WASM_EXEC_JS := wasm_exec.js
SERVER_OBJ  := main
VERSION_OBJ := version.txt
CLIENT_OBJ  := main.wasm
WORDS_OBJ   := words.txt
SERVER_TEST := server.test
CLIENT_TEST := client.test
SERVE_ARGS := $(shell grep -s -v "^\#" .env)
GENERATE_SRC := game/message/type_string.go
GO_SRC_FN = find $(1) $(foreach g,$(GENERATE_SRC),-path $g -prune -o) -name *.go -print # exclude the generated source from go sources because it is created after the version, which depends on normal source
SERVER_SRC := $(shell $(call GO_SRC_FN,cmd/server/ game/ server/ db/))
CLIENT_SRC := $(shell $(call GO_SRC_FN,cmd/ui/     game/ ui/))
EMBED_FILES ::= $(addprefix $(SERVER_EMBED_DIR)/,\
	$(TLS_CERT_FILE) \
	$(TLS_KEY_FILE) \
	$(VERSION_OBJ) \
	$(WORDS_OBJ) \
	$(STATIC_DIR) \
	$(TEMPLATE_DIR) \
	$(SQL_DIR) \
	$(addprefix $(STATIC_DIR)/,$(LICENSE_FILE) $(WASM_EXEC_JS) $(CLIENT_OBJ))) \
	$(shell if [ -d $(SERVER_EMBED_DIR) ]; then find $(SERVER_EMBED_DIR) -type f; fi)
EMBED_RESOURCES_FN = find $(PWD)/$(RESOURCES_DIR)/$(1) -type f | xargs -i{} $(LINK) {} $(SERVER_EMBED_DIR)/$(1)

$(BUILD_DIR)/$(SERVER_OBJ): $(BUILD_DIR)/$(SERVER_TEST) | $(BUILD_DIR)
	$(GO_LIST) $(GO_PACKAGES) | grep cmd/server \
		| $(GO_ARGS) xargs $(GO_BUILD) \
			-o $@

$(BUILD_DIR)/$(CLIENT_OBJ): $(BUILD_DIR)/$(CLIENT_TEST) | $(BUILD_DIR)
	$(GO_WASM_ARGS) $(GO_LIST) $(GO_PACKAGES) | grep cmd/ui \
		| $(GO_WASM_ARGS) xargs $(GO_BUILD) \
			-o $@

$(BUILD_DIR)/$(SERVER_TEST): $(EMBED_FILES) $(SERVER_SRC) $(GENERATE_SRC) | $(BUILD_DIR)
	$(GO_LIST) $(GO_PACKAGES) | grep -v ui \
		| $(GO_ARGS) xargs $(GO_TEST) $(GO_TEST_ARGS) \
		| tee $@

$(BUILD_DIR)/$(CLIENT_TEST): $(CLIENT_SRC) $(GENERATE_SRC) | $(BUILD_DIR)
	$(GO_WASM_ARGS) $(GO_LIST) $(GO_PACKAGES) | grep ui \
		| $(GO_WASM_ARGS) xargs $(GO_TEST) $(GO_TEST_ARGS) \
			-exec="$(NODE_EXEC)" \
		| tee $@

$(BUILD_DIR)/$(VERSION_OBJ): $(SERVER_SRC) $(CLIENT_SRC) $(RESOURCES_SRC) | $(BUILD_DIR)
	grep "^!" $(DOCKERIGNORE_FILE) \
		| cut -c 2- \
		| xargs -I{} find {} -type f -print \
		| sort \
		| grep -v $(SERVER_EMBED_DIR) \
		| xargs cat \
		| md5sum \
		| cut -c -32 \
		| tee $@ \
		| xargs echo $@ is

$(BUILD_DIR)/$(WORDS_OBJ): | $(BUILD_DIR)
	aspell -d en_US dump master \
		| sort \
		| uniq \
		| grep -E ^[a-z]+$$ \
		> $@

$(BUILD_DIR):
	mkdir -p $@

$(GENERATE_SRC): $(GO_MOD_FILES) | $(BUILD_DIR)/$(VERSION_OBJ)
	$(GO_INSTALL)  $(GO_PACKAGES)
	$(GO_GENERATE) $(GO_PACKAGES)

$(SERVER_EMBED_DIR)/$(VERSION_OBJ): $(BUILD_DIR)/$(VERSION_OBJ) | $(SERVER_EMBED_DIR)
	$(LINK) $< $@

$(SERVER_EMBED_DIR)/$(STATIC_DIR): $(shell find $(RESOURCES_DIR)/$(STATIC_DIR) -type f) | $(SERVER_EMBED_DIR)
	mkdir -p $@
	$(call EMBED_RESOURCES_FN,$(@F))

$(SERVER_EMBED_DIR)/$(TEMPLATE_DIR): $(shell find $(RESOURCES_DIR)/$(TEMPLATE_DIR) -type f) | $(SERVER_EMBED_DIR)
	mkdir -p $@
	$(call EMBED_RESOURCES_FN,$(@F))

$(SERVER_EMBED_DIR)/$(SQL_DIR): $(shell find $(RESOURCES_DIR)/$(SQL_DIR) -type f) | $(SERVER_EMBED_DIR)
	mkdir -p $@
	$(call EMBED_RESOURCES_FN,$(@F))

$(SERVER_EMBED_DIR)/$(TLS_CERT_FILE): $(RESOURCES_DIR)/$(TLS_CERT_FILE) | $(SERVER_EMBED_DIR)
	$(LINK) $< $@

$(SERVER_EMBED_DIR)/$(TLS_KEY_FILE): $(RESOURCES_DIR)/$(TLS_KEY_FILE) | $(SERVER_EMBED_DIR)
	$(LINK) $< $@

$(SERVER_EMBED_DIR)/$(WORDS_OBJ): $(BUILD_DIR)/$(WORDS_OBJ) | $(SERVER_EMBED_DIR)
	$(LINK) $< $@

$(SERVER_EMBED_DIR)/$(STATIC_DIR)/$(LICENSE_FILE): | $(SERVER_EMBED_DIR)/$(STATIC_DIR)
	$(LINK) $(@F) $@

$(SERVER_EMBED_DIR)/$(STATIC_DIR)/$(WASM_EXEC_JS): | $(SERVER_EMBED_DIR)/$(STATIC_DIR)
	$(COPY) $(GO_WASM_PATH)/$(@F) $@

$(SERVER_EMBED_DIR)/$(STATIC_DIR)/$(CLIENT_OBJ): $(BUILD_DIR)/$(CLIENT_OBJ) | $(SERVER_EMBED_DIR)/$(STATIC_DIR)
	$(LINK) $< $@

$(SERVER_EMBED_DIR):
	mkdir -p $@

$(addprefix $(RESOURCES_DIR)/,$(TLS_CERT_FILE) $(TLS_KEY_FILE)):
	touch $@

all: $(BUILD_DIR)/$(SERVER_OBJ)

test: $(BUILD_DIR)/$(SERVER_TEST)

clean:
	rm -rf $(BUILD_DIR) $(SERVER_EMBED_DIR) $(GENERATE_SRC)

serve: all
	$(SERVE_ARGS) $<

serve-tcp: all
	sudo setcap cap_net_bind_service=+ep $<
	$(SERVE_ARGS) HTTP_PORT=80 HTTPS_PORT=443 $<

# list variables: https://stackoverflow.com/a/7144684/1330346
# make -pn | grep -A1 "^# makefile"| grep -v "^#\|^--" | sort | uniq

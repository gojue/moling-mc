include variables.mk
include functions.mk

.PHONY: all | env
all: clean test build
	@echo $(shell date)

.ONESHELL:
SHELL = /bin/bash

.PHONY: env
env:
	@echo ---------------------------------------
	@echo "eCapture Makefile Environment:"
	@echo ---------------------------------------
	@echo ----------------[ from args ]---------------
	@echo "SNAPSHOT_VERSION         $(SNAPSHOT_VERSION)"
	@echo ---------------------------------------
	@echo "OS_NAME                  $(OS_NAME)"
	@echo "OS_ARCH                  $(OS_ARCH)"
	@echo "GO_VERSION               $(GO_VERSION)"
	@echo ---------------------------------------
	@echo "CMD_GIT                  $(CMD_GIT)"
	@echo "CMD_GO                   $(CMD_GO)"
	@echo "CMD_INSTALL              $(CMD_INSTALL)"
	@echo "CMD_MD5                  $(CMD_MD5)"
	@echo ---------------------------------------
	@echo "VERSION_NUM              $(VERSION_NUM)"
	@echo "LAST_GIT_TAG             $(LAST_GIT_TAG)"
	@echo "GOARCH                   $(GOARCH)"
	@echo ---------------------------------------


.PHONY: help
help:
	@echo "# environment"
	@echo "    $$ make env					# show makefile environment/variables"
	@echo ""
	@echo "# build"
	@echo "    $$ make all					# build ecapture"
	@echo ""
	@echo "# clean"
	@echo "    $$ make clean				# wipe ./bin/ ./user/bytecode/ ./assets/"
	@echo ""
	@echo "# test"

.PHONY: clean build

.PHONY: clean
clean:
	$(CMD_RM) -f $(OUT_BIN)*

.PHONY: build
build:
	$(call gobuild)

.PHONY: dev
dev: clean
	$(call allow-override, TARGET_OSS, $(OS_NAME))
	$(call allow-override, TARGET_ARCHS, $(OS_ARCH))
	$(call gobuild)

# Format the code
.PHONY: format
format:
	@echo "  ->  Formatting code"

.PHONY: test
test:
	go test -v -race ./...

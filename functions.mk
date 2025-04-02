define allow-override
  $(if $(or $(findstring environment,$(origin $(1))),\
            $(findstring command line,$(origin $(1)))),,\
    $(eval $(1) = $(2)))
endef

# TARGET_OS , TARGET_ARCH
define gobuild
	CGO_ENABLED=0 \
	GOOS=$(1) GOARCH=$(2) \
	$(eval OUT_BIN_SUFFIX=$(if $(filter $(1),windows),.exe,)) \
	$(CMD_GO) build -trimpath -mod=readonly -ldflags "-w -s -X 'github.com/gojue/moling/cli/cmd.GitVersion=$(1)_$(2)_$(VERSION_NUM)'" -o $(OUT_BIN)$(OUT_BIN_SUFFIX)
	$(CMD_FILE) $(OUT_BIN)$(OUT_BIN_SUFFIX)
endef


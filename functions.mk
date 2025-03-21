define allow-override
  $(if $(or $(findstring environment,$(origin $(1))),\
            $(findstring command line,$(origin $(1)))),,\
    $(eval $(1) = $(2)))
endef

define gobuild
	$(foreach TARGET_OS_NAME, $(TARGET_OSS),\
		$(foreach TARGET_OS_ARCH, $(TARGET_ARCHS),\
			CGO_ENABLED=0 \
			GOOS=$(TARGET_OS_NAME) GOARCH=$(TARGET_OS_ARCH) \
			$(CMD_GO) build -trimpath -mod=readonly -ldflags "-w -s -X 'github.com/gojue/moling/cli/cmd.GitVersion=$(TARGET_OS_NAME)-$(TARGET_OS_ARCH)-$(VERSION_NUM)'" -o $(OUT_BIN)-$(TARGET_OS_NAME)-$(TARGET_OS_ARCH)
			$(CMD_FILE) $(OUT_BIN)-$(TARGET_OS_NAME)-$(TARGET_OS_ARCH)
		)
	)
endef


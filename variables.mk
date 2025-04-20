CMD_MAKE = make
CMD_TR ?= tr
CMD_CUT ?= cut
CMD_AWK ?= awk
CMD_SED ?= sed
CMD_FILE ?= file
CMD_GIT ?= git
CMD_CLANG ?= clang
CMD_RM ?= rm
CMD_INSTALL ?= install
CMD_MKDIR ?= mkdir
CMD_TOUCH ?= touch
CMD_GO ?= go
CMD_GREP ?= grep
CMD_CAT ?= cat
CMD_MD5 ?= md5sum
CMD_TAR ?= tar
CMD_CHECKSUM ?= sha256sum
CMD_GITHUB ?= gh
CMD_MV ?= mv
CMD_CP ?= cp
CMD_CD ?= cd
CMD_ECHO ?= echo


#
# tools version
#

GO_VERSION = $(shell $(CMD_GO) version 2>/dev/null | $(CMD_AWK) '{print $$3}' | $(CMD_SED) 's:go::g' | $(CMD_CUT) -d. -f1,2)
GO_VERSION_MAJ = $(shell $(CMD_ECHO) $(GO_VERSION) | $(CMD_CUT) -d'.' -f1)
GO_VERSION_MIN = $(shell $(CMD_ECHO) $(GO_VERSION) | $(CMD_CUT) -d'.' -f2)

# tags date info
VERSION_NUM ?= v0.0.0
NOW_DATE := $(shell date +"%Y-%m-%d %H:%M:%S")
TAG_COMMIT := $(shell git rev-list --abbrev-commit --tags --max-count=1)
TAG := $(shell git describe --abbrev=0 --tags ${TAG_COMMIT} 2>/dev/null || true)
COMMIT := $(shell git rev-parse --short HEAD)
DATE := $(shell git log -1 --format=%cd --date=format:"%Y %m %d")
LAST_GIT_TAG := $(COMMIT)_$(NOW_DATE)

ifndef SNAPSHOT_VERSION
	VERSION_NUM = $(LAST_GIT_TAG)
else
	VERSION_NUM = $(SNAPSHOT_VERSION)_$(NOW_DATE)
endif

#
# environment
#
#SNAPSHOT_VERSION ?= $(shell git rev-parse HEAD)
BUILD_DATE := $(shell date +%Y-%m-%d)
TARGET_TAG :=
OS_NAME ?= $(shell uname -s|tr 'A-Z' 'a-z')
OS_ARCH ?= $(shell uname -m)
OS_VERSION_SHORT := $(shell uname -r | cut -d'-' -f 1)
TARGET_OS ?= $(OS_NAME)
TARGET_ARCH ?= $(if $(filter x86_64,$(OS_ARCH)),amd64,arm64)
OUT_BIN := bin/moling_mc
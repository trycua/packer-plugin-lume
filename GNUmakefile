NAME=lume
BINARY=packer-plugin-${NAME}

COUNT?=1
TEST?=$(shell go list ./...)
HASHICORP_PACKER_PLUGIN_SDK_VERSION?=$(shell go list -m github.com/hashicorp/packer-plugin-sdk | cut -d " " -f2)
PLUGIN_FQN=$(shell grep -E '^module' <go.mod | sed -E 's/module \s*//')

.PHONY: dev

build:
	@go build -o ${BINARY}

dev:
	go build -ldflags="-X '${PLUGIN_FQN}/version.Version=0.0.1'" -o ${BINARY}
	packer plugins install --path ${BINARY} "$(shell echo "${PLUGIN_FQN}" | sed 's/packer-plugin-//')"

test:
	@go test -race -count $(COUNT) $(TEST) -timeout=3m

install-packer-sdc: ## Install packer sofware development command
	@go install github.com/hashicorp/packer-plugin-sdk/cmd/packer-sdc@${HASHICORP_PACKER_PLUGIN_SDK_VERSION}

plugin-check: install-packer-sdc build
	@packer-sdc plugin-check ${BINARY}

testacc: dev
	@PACKER_ACC=1 go test -count $(COUNT) -v $(TEST) -timeout=120m

generate: install-packer-sdc
	@go generate ./...
	@rm -rf .docs
	@packer-sdc renderdocs -src docs -partials docs-partials/ -dst .docs/
	@./.web-docs/scripts/compile-to-webdocs.sh "." ".docs" ".web-docs" "hashicorp"
	@rm -r ".docs"

# NAME=lume
# ROOT_DIR:=$(dir $(realpath $(lastword $(MAKEFILE_LIST))))
# BUILD_DIR=build
# PLUGIN_DIR=${BUILD_DIR}/plugins
# BINARY=packer-plugin-${NAME}_v0.0.0_x5.0_linux_amd64
# # https://github.com/hashicorp/packer-plugin-sdk/issues/187
# HASHICORP_PACKER_PLUGIN_SDK_VERSION?="v0.5.2"
# PLUGIN_FQN=$(shell grep -E '^module' <go.mod | sed -E 's/module \s*//')
# PLUGIN_PATH=.

# .PHONY: build
# build:
# 	@mkdir -p ${PLUGIN_DIR}
# 	@go build -ldflags="-X '${PLUGIN_FQN}/main.Version=$(shell git describe --tags --abbrev=0)'" -o ${PLUGIN_DIR}/${BINARY} ${PLUGIN_PATH}
# 	@sha256sum < ${PLUGIN_DIR}/${BINARY} > ${PLUGIN_DIR}/${BINARY}_SHA256SUM

# .PHONY: clean
# clean:
# 	@rm -rf ${BUILD_DIR}

# .PHONY: start
# start: build
# 	PACKER_PLUGIN_PATH=${ROOT_DIR}${BUILD_DIR} PACKER_LOG=1 PACKER_LOG_PATH=packer.log packer build empty.pkr.hcl
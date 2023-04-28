.DEFAULT_GOAL := all

SHELL := /usr/bin/env bash -o pipefail
DOCKER ?= docker
DOCKER_ORG ?= bufbuild
DOCKER_BUILD_EXTRA_ARGS ?=

GO_TEST_FLAGS ?= -race -count=1

BUF ?= /Users/pmitchell/.cache/buf/Darwin/arm64/gobin/buf
BUF_PLUGIN_PUSH_ARGS ?=

# Specify a space separated list of plugin name (and optional version) to just build/test individual plugins.
# For example:
# $ make PLUGINS="connect-go connect-web" # builds all versions of connect-go and connect-web plugins
# $ make PLUGINS="connect-go:v0.4.0"      # builds connect-go v0.4.0 plugin
# $ make PLUGINS="bufbuild/connect-go"    # can use optional prefix of the org
export PLUGINS ?=

PLUGIN_YAML_FILES := $(shell PLUGINS="$(PLUGINS)" go run ./internal/cmd/dependency-order -relative . 2>/dev/null)

.PHONY: all
all: build
	@if [[ -z "${PLUGINS}" ]]; then \
		echo "No plugins specified to build with PLUGINS env var."; \
		echo "See Makefile for example PLUGINS env var usage."; \
		echo "To build all plugins (will take a long time), build with 'make PLUGINS=all'."; \
	fi

.PHONY: build
build:
	@go run ./internal/cmd/dockerbuild -org "$(DOCKER_ORG)" -- $(DOCKER_BUILD_EXTRA_ARGS)

.PHONY: dockerpush
dockerpush:
	@go run ./internal/cmd/dockerpush -org "$(DOCKER_ORG)"

.PHONY: test
test: build
	go test $(GO_TEST_FLAGS) ./...

.PHONY: push
push: build
	for plugin in $(PLUGIN_YAML_FILES); do \
		plugin_dir=`dirname $${plugin}`; \
		PLUGIN_FULL_NAME=`yq '.name' $${plugin_dir}/buf.plugin.yaml`; \
		PLUGIN_OWNER=`echo "$${PLUGIN_FULL_NAME}" | cut -d '/' -f 2`; \
		PLUGIN_NAME=`echo "$${PLUGIN_FULL_NAME}" | cut -d '/' -f 3-`; \
		PLUGIN_VERSION=`yq '.plugin_version' $${plugin_dir}/buf.plugin.yaml`; \
		echo "Pushing plugin: $${plugin}"; \
		if [[ "$(DOCKER_ORG)" = "ghcr.io/bufbuild" ]]; then \
			$(DOCKER) pull $(DOCKER_ORG)/plugins-$${PLUGIN_OWNER}-$${PLUGIN_NAME}:$${PLUGIN_VERSION} || exit 1; \
		fi; \
		$(BUF) alpha plugin push $${plugin_dir} $(BUF_PLUGIN_PUSH_ARGS) --image $(DOCKER_ORG)/plugins-$${PLUGIN_OWNER}-$${PLUGIN_NAME}:$${PLUGIN_VERSION} || exit 1; \
	done

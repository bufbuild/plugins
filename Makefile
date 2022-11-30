.DEFAULT_GOAL := all

SHELL := /usr/bin/env bash -o pipefail
DOCKER ?= docker
DOCKER_ORG ?= bufbuild
DOCKER_BUILD_ARGS ?= buildx build
DOCKER_BUILD_EXTRA_ARGS ?=

GO_TEST_FLAGS ?= -race -count=1

BUF ?= buf
BUF_PLUGIN_PUSH_ARGS ?=

# Specify a space separated list of plugin name (and optional version) to just build/test individual plugins.
# For example:
# $ make PLUGINS="connect-go connect-web" # builds all versions of connect-go and connect-web plugins
# $ make PLUGINS="connect-go:v0.4.0"      # builds connect-go v0.4.0 plugin
# $ make PLUGINS="bufbuild/connect-go"    # can use optional prefix of the org
PLUGINS ?=

PLUGIN_YAML_FILES := $(shell PLUGINS="$(PLUGINS)" go run ./cmd/dependency-order .)
PLUGIN_IMAGES := $(patsubst %/buf.plugin.yaml,.build/plugin/%/image,$(PLUGIN_YAML_FILES))

.PHONY: all
all: build

.PHONY: build
build: $(BASE_IMAGES) $(PLUGIN_IMAGES)

.PHONY: clean
clean:
	@rm -rf .build

.PHONY: test
test: build
	go test $(GO_TEST_FLAGS) ./...

.build/plugin/%/image: %/Dockerfile %/buf.plugin.yaml
	PLUGIN_FULL_NAME=$(shell yq '.name' $*/buf.plugin.yaml); \
	PLUGIN_OWNER=`echo "$${PLUGIN_FULL_NAME}" | cut -d '/' -f 2`; \
	PLUGIN_NAME=`echo "$${PLUGIN_FULL_NAME}" | cut -d '/' -f 3-`; \
	PLUGIN_VERSION=$(shell yq '.plugin_version' $*/buf.plugin.yaml); \
	PLUGIN_LICENSE="$(shell yq '.spdx_license_id' $*/buf.plugin.yaml)"; \
	test -n "$${PLUGIN_NAME}" -a -n "$${PLUGIN_VERSION}" && \
	if [[ "$(DOCKER_ORG)" = "ghcr.io/bufbuild" ]]; then \
		$(DOCKER) pull $(DOCKER_ORG)/plugins-$${PLUGIN_OWNER}-$${PLUGIN_NAME}:$${PLUGIN_VERSION} || :; \
	fi; \
	touch $<; \
	$(DOCKER) $(DOCKER_BUILD_ARGS) \
		$(DOCKER_BUILD_EXTRA_ARGS) \
		--label build.buf.plugins.config.owner=$${PLUGIN_OWNER} \
		--label build.buf.plugins.config.name=$${PLUGIN_NAME} \
		--label build.buf.plugins.config.version=$${PLUGIN_VERSION} \
		--label org.opencontainers.image.source=https://github.com/bufbuild/plugins \
		--label "org.opencontainers.image.licenses=$${PLUGIN_LICENSE}" \
		-t $(DOCKER_ORG)/plugins-$${PLUGIN_OWNER}-$${PLUGIN_NAME}:$${PLUGIN_VERSION} \
		$(<D)
	@mkdir -p $(dir $@) && touch $@

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

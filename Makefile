.DEFAULT_GOAL := all

SHELL := /usr/bin/env bash -o pipefail
DOCKER ?= docker
DOCKER_ORG ?= bufbuild
DOCKER_BUILD_ARGS ?= buildx build --build-arg DOCKER_ORG="$(DOCKER_ORG)"
DOCKER_BUILD_EXTRA_ARGS ?=
DOCKER_READ_CACHE_ORG ?=
DOCKER_WRITE_CACHE_ORG ?=

GO_TEST_FLAGS ?= -race -count=1

BUF ?= buf
BUF_PLUGIN_PUSH_ARGS ?=

BASE_DOCKERFILES := $(shell go run ./cmd/dependency-order -dockerfile -base .)
BASE_IMAGES := $(patsubst %/Dockerfile,.build/base/%/image,$(BASE_DOCKERFILES))
PLUGIN_YAML_FILES := $(shell go run ./cmd/dependency-order .)
PLUGIN_IMAGES := $(patsubst %/buf.plugin.yaml,.build/plugin/%/image,$(PLUGIN_YAML_FILES))

.PHONY: all
all: build

.PHONY: build
build: $(BASE_IMAGES) $(PLUGIN_IMAGES)

.PHONY: clean
clean:
	@rm -rf .build

.PHONY: test
test:
	go test $(GO_TEST_FLAGS) ./...

.build/base/library/%/base-build/image: library/%/base-build/Dockerfile
	CACHE_ARGS=""; \
	if [[ -n "$(DOCKER_READ_CACHE_ORG)" ]]; then \
		CACHE_ARGS="$${CACHE_ARGS} --cache-from type=registry,ref=$(DOCKER_READ_CACHE_ORG)/plugins-$*-base-build:buildcache"; \
		$(DOCKER) pull $(DOCKER_ORG)/plugins-$*-base-build:latest || :; \
	fi; \
	if [[ -n "$(DOCKER_WRITE_CACHE_ORG)" ]]; then \
		CACHE_ARGS="$${CACHE_ARGS} --cache-to type=registry,mode=max,ref=$(DOCKER_WRITE_CACHE_ORG)/plugins-$*-base-build:buildcache"; \
	fi; \
	$(DOCKER) $(DOCKER_BUILD_ARGS) $(DOCKER_BUILD_EXTRA_ARGS)$${CACHE_ARGS} -t $(DOCKER_ORG)/plugins-$*-base-build $(<D)
	@mkdir -p $(dir $@) && touch $@

.build/base/library/grpc/v%/base/image: library/grpc/v%/base/Dockerfile
	VERSION=v$(shell basename $*); \
	CACHE_ARGS=""; \
	if [[ -n "$(DOCKER_READ_CACHE_ORG)" ]]; then \
		CACHE_ARGS="$${CACHE_ARGS} --cache-from type=registry,ref=$(DOCKER_READ_CACHE_ORG)/plugins-grpc-base:$${VERSION}-buildcache"; \
		$(DOCKER) pull $(DOCKER_ORG)/plugins-grpc-base:$${VERSION} || :; \
	fi; \
	if [[ -n "$(DOCKER_WRITE_CACHE_ORG)" ]]; then \
		CACHE_ARGS="$${CACHE_ARGS} --cache-to type=registry,mode=max,ref=$(DOCKER_WRITE_CACHE_ORG)/plugins-grpc-base:$${VERSION}-buildcache"; \
	fi; \
	$(DOCKER) $(DOCKER_BUILD_ARGS) $(DOCKER_BUILD_EXTRA_ARGS)$${CACHE_ARGS} -t $(DOCKER_ORG)/plugins-grpc-base:$${VERSION} $(<D)
	@mkdir -p $(dir $@) && touch $@

.build/base/library/protoc/v%/base/image: library/protoc/v%/base/Dockerfile
	VERSION=v$(shell basename $*); \
	CACHE_ARGS=""; \
	if [[ -n "$(DOCKER_READ_CACHE_ORG)" ]]; then \
		CACHE_ARGS="$${CACHE_ARGS} --cache-from type=registry,ref=$(DOCKER_READ_CACHE_ORG)/plugins-protoc-base:$${VERSION}-buildcache"; \
		$(DOCKER) pull $(DOCKER_ORG)/plugins-protoc-base:$${VERSION} || :; \
	fi; \
	if [[ -n "$(DOCKER_WRITE_CACHE_ORG)" ]]; then \
		CACHE_ARGS="$${CACHE_ARGS} --cache-to type=registry,mode=max,ref=$(DOCKER_WRITE_CACHE_ORG)/plugins-protoc-base:$${VERSION}-buildcache"; \
	fi; \
	$(DOCKER) $(DOCKER_BUILD_ARGS) $(DOCKER_BUILD_EXTRA_ARGS)$${CACHE_ARGS} -t $(DOCKER_ORG)/plugins-protoc-base:$${VERSION} $(<D)
	@mkdir -p $(dir $@) && touch $@

.build/plugin/%/image: %/Dockerfile %/buf.plugin.yaml $(BASE_IMAGES)
	PLUGIN_FULL_NAME=$(shell yq '.name' $*/buf.plugin.yaml); \
	PLUGIN_OWNER=`echo "$${PLUGIN_FULL_NAME}" | cut -d '/' -f 2`; \
	PLUGIN_NAME=`echo "$${PLUGIN_FULL_NAME}" | cut -d '/' -f 3-`; \
	PLUGIN_VERSION=$(shell yq '.plugin_version' $*/buf.plugin.yaml); \
	CACHE_ARGS=""; \
	if [[ -n "$(DOCKER_READ_CACHE_ORG)" ]]; then \
		CACHE_ARGS="$${CACHE_ARGS} --cache-from type=registry,ref=$(DOCKER_READ_CACHE_ORG)/plugins-$${PLUGIN_OWNER}-$${PLUGIN_NAME}:$${PLUGIN_VERSION}-buildcache"; \
		$(DOCKER) pull $(DOCKER_ORG)/plugins-$${PLUGIN_OWNER}-$${PLUGIN_NAME}:$${PLUGIN_VERSION} || :; \
	fi; \
	if [[ -n "$(DOCKER_WRITE_CACHE_ORG)" ]]; then \
		CACHE_ARGS="$${CACHE_ARGS} --cache-to type=registry,mode=max,ref=$(DOCKER_WRITE_CACHE_ORG)/plugins-$${PLUGIN_OWNER}-$${PLUGIN_NAME}:$${PLUGIN_VERSION}-buildcache"; \
	fi; \
	test -n "$${PLUGIN_NAME}" -a -n "$${PLUGIN_VERSION}" && \
	$(DOCKER) $(DOCKER_BUILD_ARGS) \
		$(DOCKER_BUILD_EXTRA_ARGS)$${CACHE_ARGS} \
		--build-arg PLUGIN_VERSION=$${PLUGIN_VERSION} \
		--label build.buf.plugins.config.owner=$${PLUGIN_OWNER} \
		--label build.buf.plugins.config.name=$${PLUGIN_NAME} \
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
		if [[ "$(DOCKER_ORG)" != "bufbuild" ]]; then \
			$(DOCKER) pull $(DOCKER_ORG)/plugins-$${PLUGIN_OWNER}-$${PLUGIN_NAME}:$${PLUGIN_VERSION} || exit 1; \
			$(BUF) alpha plugin push $${plugin_dir} --image $(DOCKER_ORG)/plugins-$${PLUGIN_OWNER}-$${PLUGIN_NAME}:$${PLUGIN_VERSION} || exit 1; \
		else \
			if [[ -n "$(DOCKER_READ_CACHE_ORG)" ]]; then \
				CACHE_ARGS=" --cache-from $(DOCKER_READ_CACHE_ORG)/plugins-$${PLUGIN_OWNER}-$${PLUGIN_NAME}:$${PLUGIN_VERSION}-buildcache"; \
			fi; \
			$(BUF) alpha plugin push $${plugin_dir} $(BUF_PLUGIN_PUSH_ARGS)$${CACHE_ARGS} --build-arg DOCKER_ORG="$(DOCKER_ORG)" || exit 1; \
		fi; \
	done

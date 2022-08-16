.DEFAULT_GOAL := all

DOCKER ?= docker
DOCKER_BUILD_ARGS ?= buildx build

GO_TEST_FLAGS ?= -race -count=1

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
	$(DOCKER) $(DOCKER_BUILD_ARGS) -t bufbuild/plugins-$*-base-build $(<D)
	@mkdir -p $(dir $@) && touch $@

.build/base/library/grpc/v%/base/image: library/grpc/v%/base/Dockerfile
	GRPC_VERSION=v$(shell basename $*); \
	$(DOCKER) $(DOCKER_BUILD_ARGS) --build-arg GRPC_VERSION=$${GRPC_VERSION} -t bufbuild/plugins-grpc-base:$${GRPC_VERSION} $(<D)
	@mkdir -p $(dir $@) && touch $@

.build/base/library/protoc/v%/base/image: library/protoc/v%/base/Dockerfile
	PROTOC_VERSION=v$(shell basename $*); \
	$(DOCKER) $(DOCKER_BUILD_ARGS) --build-arg PROTOC_VERSION=$${PROTOC_VERSION} -t bufbuild/plugins-protoc-base:$${PROTOC_VERSION} $(<D)
	@mkdir -p $(dir $@) && touch $@

.build/plugin/%/image: %/Dockerfile %/buf.plugin.yaml $(BASE_IMAGES)
	PLUGIN_FULL_NAME=$(shell yq '.name' $*/buf.plugin.yaml); \
	PLUGIN_OWNER=`echo "$${PLUGIN_FULL_NAME}" | cut -d '/' -f 2`; \
	PLUGIN_NAME=`echo "$${PLUGIN_FULL_NAME}" | cut -d '/' -f 3-`; \
	PLUGIN_VERSION=$(shell yq '.plugin_version' $*/buf.plugin.yaml); \
	test -n "$${PLUGIN_NAME}" -a -n "$${PLUGIN_VERSION}" && \
	$(DOCKER) $(DOCKER_BUILD_ARGS) --build-arg PLUGIN_VERSION=$${PLUGIN_VERSION} -t bufbuild/plugins-$${PLUGIN_OWNER}-$${PLUGIN_NAME}:$${PLUGIN_VERSION} $(<D)
	@mkdir -p $(dir $@) && touch $@

.PHONY: push
push: build
	@for plugin in $(PLUGIN_YAML_FILES); do \
		plugin_dir=$$(dirname $${plugin}); \
		echo "Pushing plugin: $${plugin}"; \
		buf alpha plugin push $${plugin_dir} $(BUF_PLUGIN_PUSH_ARGS) || exit 1; \
	done

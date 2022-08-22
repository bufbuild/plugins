# syntax=docker/dockerfile:1.4
ARG DOCKER_ORG=bufbuild
ARG BASE_BUILD_VERSION=latest
FROM ${DOCKER_ORG}/plugins-protoc-base-build:${BASE_BUILD_VERSION}

ARG VERSION
RUN test -n "${VERSION}"

RUN git clone https://github.com/protocolbuffers/protobuf --depth 1 --branch ${VERSION} --recursive
WORKDIR /build/protobuf/
COPY plugins/* /build/plugins/
RUN ["ln", "-s", "/build/plugins", "."]
RUN ["/bin/bash", "-c", "bazel build $(bazel query 'kind(cc_binary, //plugins/...)')"]

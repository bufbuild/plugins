# syntax=docker/dockerfile:1.4
ARG DOCKER_ORG=bufbuild
ARG BASE_BUILD_VERSION=latest
FROM ${DOCKER_ORG}/plugins-protoc-base-build:${BASE_BUILD_VERSION}

RUN git clone https://github.com/protocolbuffers/protobuf --depth 1 --branch v21.4 --recursive
WORKDIR /build/protobuf/
RUN ln -s /build/plugins .
RUN bazel build $(bazel query 'kind(cc_binary, //plugins/...)')

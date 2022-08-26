# syntax=docker/dockerfile:1.4
ARG DOCKER_ORG=bufbuild
ARG BASE_BUILD_VERSION=latest
FROM ${DOCKER_ORG}/plugins-protoc-base-build:${BASE_BUILD_VERSION}

RUN git clone https://github.com/protocolbuffers/protobuf --depth 1 --branch v3.20.1 --recursive
WORKDIR /build/protobuf/
COPY --link plugins/* /build/plugins/
RUN ln -s /build/plugins .
RUN bazel build $(bazel query 'kind(cc_binary, //plugins/...)')

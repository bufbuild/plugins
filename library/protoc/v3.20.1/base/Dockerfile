# syntax=docker/dockerfile:1.4
FROM bufbuild/plugins-protoc-base-build:latest

ARG PROTOC_VERSION
RUN test -n "${PROTOC_VERSION}"

RUN git clone https://github.com/protocolbuffers/protobuf --depth 1 --branch ${PROTOC_VERSION} --recursive
WORKDIR /build/protobuf/
COPY plugins/* /build/plugins/
RUN ["ln", "-s", "/build/plugins", "."]
RUN ["/bin/bash", "-c", "bazel build $(bazel query 'kind(cc_binary, //plugins/...)')"]

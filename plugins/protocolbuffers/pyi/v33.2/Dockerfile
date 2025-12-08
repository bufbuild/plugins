# syntax=docker/dockerfile:1.19
FROM debian:bookworm-20251117 AS build

ARG TARGETARCH
ARG BAZEL_OPTS="--host_jvm_args=-Djava.net.preferIPv4Stack=true"

RUN apt-get update \
 && apt-get install -y curl git cmake build-essential autoconf clang libc++-dev libtool pkg-config unzip zip
RUN curl -fsSL -o /usr/local/bin/bazelisk https://github.com/bazelbuild/bazelisk/releases/download/v1.25.0/bazelisk-linux-${TARGETARCH} \
 && chmod +x /usr/local/bin/bazelisk \
 && mkdir /build \
 && chown nobody:nogroup /build \
 && usermod --home /build nobody

USER nobody
WORKDIR /build
RUN curl -fsSL -o protoc.tar.gz https://github.com/protocolbuffers/protobuf/releases/download/v33.2/protobuf-33.2.tar.gz \
 && tar --strip-components=1 -zxf protoc.tar.gz \
 && rm protoc.tar.gz
RUN bazelisk ${BAZEL_OPTS} build '//:protoc_lib'
COPY BUILD pyi.cc plugins/
RUN bazelisk ${BAZEL_OPTS} build '//plugins:protoc-gen-pyi.stripped'

FROM gcr.io/distroless/cc-debian12:latest@sha256:0c8eac8ea42a167255d03c3ba6dfad2989c15427ed93d16c53ef9706ea4691df AS base

FROM scratch
COPY --from=base / /
COPY --from=build --link --chmod=0755 /build/bazel-bin/plugins/protoc-gen-pyi .
USER nobody
ENTRYPOINT ["/protoc-gen-pyi"]

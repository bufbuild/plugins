# syntax=docker/dockerfile:1.12
FROM debian:bookworm-20250224 AS build

ENV USE_BAZEL_VERSION=7.4.1
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
RUN curl -fsSL -o protoc.tar.gz https://github.com/protocolbuffers/protobuf/releases/download/v25.6/protobuf-25.6.tar.gz \
 && tar --strip-components=1 -zxf protoc.tar.gz \
 && rm protoc.tar.gz
RUN bazelisk ${BAZEL_OPTS} build '//:protoc_lib'
COPY --link BUILD php.cc plugins/
RUN bazelisk ${BAZEL_OPTS} build '//plugins:protoc-gen-php.stripped'

FROM gcr.io/distroless/cc-debian12:latest@sha256:b7550f0b15838de14c564337eef2b804ba593ae55d81ca855421bd52f19bb480 AS base

FROM scratch
COPY --from=base / /
COPY --from=build --link --chmod=0755 /build/bazel-bin/plugins/protoc-gen-php .
USER nobody
ENTRYPOINT ["/protoc-gen-php"]

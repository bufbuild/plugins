# syntax=docker/dockerfile:1.26
FROM debian:trixie-20260824@sha256:f324c7ff54321e8d9c588493a20244965938ce0aa50bbd1022d38010e9ffc4b1 AS build

ARG TARGETARCH
ARG BAZEL_OPTS="--host_jvm_args=-Djava.net.preferIPv4Stack=true"

RUN apt-get update \
 && apt-get install -y curl git cmake build-essential autoconf clang libc++-dev libtool pkg-config unzip zip
RUN curl -fsSL -o /usr/local/bin/bazelisk https://github.com/bazelbuild/bazelisk/releases/download/v1.28.1/bazelisk-linux-${TARGETARCH} \
 && chmod +x /usr/local/bin/bazelisk \
 && mkdir /build \
 && chown nobody:nogroup /build \
 && usermod --home /build nobody

USER nobody
WORKDIR /build
RUN curl -fsSL -o protoc.tar.gz https://github.com/protocolbuffers/protobuf/releases/download/v36.2/protobuf-36.2.tar.gz \
 && tar --strip-components=1 -zxf protoc.tar.gz \
 && rm protoc.tar.gz
RUN bazelisk ${BAZEL_OPTS} build '//:protoc_lib'
COPY BUILD pyi.cc plugins/
RUN bazelisk ${BAZEL_OPTS} build '//plugins:protoc-gen-pyi.stripped'

FROM gcr.io/distroless/cc-debian13:latest@sha256:4594d59540d1948417f6ca2829ddd9294493a7c68b7528f4dd459de7f203a750 AS base

FROM scratch
COPY --from=base / /
COPY --from=build --link --chmod=0755 /build/bazel-bin/plugins/protoc-gen-pyi .
USER nobody
ENTRYPOINT ["/protoc-gen-pyi"]

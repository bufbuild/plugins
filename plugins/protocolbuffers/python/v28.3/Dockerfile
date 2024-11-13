# syntax=docker/dockerfile:1.10
FROM debian:bookworm-20241016 AS build

ARG TARGETARCH
ARG BAZEL_OPTS="--host_jvm_args=-Djava.net.preferIPv4Stack=true"

RUN apt-get update \
 && apt-get install -y curl git cmake build-essential autoconf clang libc++-dev libtool pkg-config unzip zip
RUN curl -fsSL -o /usr/local/bin/bazelisk https://github.com/bazelbuild/bazelisk/releases/download/v1.23.0/bazelisk-linux-${TARGETARCH} \
 && chmod +x /usr/local/bin/bazelisk \
 && mkdir /build \
 && chown nobody:nogroup /build \
 && usermod --home /build nobody

USER nobody
WORKDIR /build
RUN curl -fsSL -o protoc.tar.gz https://github.com/protocolbuffers/protobuf/releases/download/v28.3/protobuf-28.3.tar.gz \
 && tar --strip-components=1 -zxf protoc.tar.gz \
 && rm protoc.tar.gz
RUN bazelisk ${BAZEL_OPTS} build '//:protoc_lib'
COPY --link BUILD python.cc plugins/
RUN bazelisk ${BAZEL_OPTS} build '//plugins:protoc-gen-python.stripped'

FROM gcr.io/distroless/cc-debian12:latest@sha256:3310655aac0d85eb9d579792387af1ff3eb7a1667823478be58020ab0e0d97a8 AS base

FROM scratch
COPY --from=base --link / /
COPY --from=build --link --chmod=0755 /build/bazel-bin/plugins/protoc-gen-python .
USER nobody
ENTRYPOINT ["/protoc-gen-python"]

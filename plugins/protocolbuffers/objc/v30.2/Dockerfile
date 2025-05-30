# syntax=docker/dockerfile:1.12
FROM debian:bookworm-20250317 AS build

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
RUN curl -fsSL -o protoc.tar.gz https://github.com/protocolbuffers/protobuf/releases/download/v30.2/protobuf-30.2.tar.gz \
 && tar --strip-components=1 -zxf protoc.tar.gz \
 && rm protoc.tar.gz
RUN bazelisk ${BAZEL_OPTS} build '//:protoc_lib'
COPY --link BUILD objectivec.cc plugins/
RUN bazelisk ${BAZEL_OPTS} build '//plugins:protoc-gen-objectivec.stripped'

FROM gcr.io/distroless/cc-debian12:latest@sha256:85dac24dd2f03e841d986d5ed967385d3a721dcd9dbd21b602ddd82437f364c9 AS base

FROM scratch
COPY --from=base --link / /
COPY --from=build --link --chmod=0755 /build/bazel-bin/plugins/protoc-gen-objectivec .
USER nobody
ENTRYPOINT ["/protoc-gen-objectivec"]

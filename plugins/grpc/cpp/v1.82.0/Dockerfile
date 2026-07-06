# syntax=docker/dockerfile:1.24
FROM debian:trixie-20260623@sha256:d07d1b51c39f51188e60be9b64e6bf769fa94e187f092bc32b91305cfa34ba5a AS build

ARG TARGETARCH
ARG BAZEL_OPTS="--host_jvm_args=-Djava.net.preferIPv4Stack=true"

RUN apt-get update \
 && apt-get install -y curl git cmake build-essential autoconf clang libc++-dev libtool pkg-config unzip zip
RUN curl -fsSL -o /usr/local/bin/bazelisk https://github.com/bazelbuild/bazelisk/releases/download/v1.26.0/bazelisk-linux-${TARGETARCH} \
 && chmod +x /usr/local/bin/bazelisk \
 && mkdir /build \
 && chown nobody:nogroup /build \
 && usermod --home /build nobody

USER nobody
WORKDIR /build

RUN git clone --depth 1 --branch v1.82.0 https://github.com/grpc/grpc
WORKDIR /build/grpc
RUN bazelisk ${BAZEL_OPTS} build //src/compiler:grpc_plugin_support
RUN bazelisk ${BAZEL_OPTS} build //src/compiler:grpc_cpp_plugin

FROM gcr.io/distroless/cc-debian13:latest@sha256:a017e74bd2a12d98342dbecd33d121d2b160415ed777573dc1808969e989d94d AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 /build/grpc/bazel-bin/src/compiler/grpc_cpp_plugin_binary .
USER nobody
ENTRYPOINT ["/grpc_cpp_plugin_binary"]

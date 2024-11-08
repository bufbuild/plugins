# syntax=docker/dockerfile:1.10
FROM debian:bookworm-20241016 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl git cmake build-essential autoconf clang libc++-dev libtool pkg-config unzip zip
RUN curl -fsSL -o /usr/local/bin/bazelisk https://github.com/bazelbuild/bazelisk/releases/download/v1.23.0/bazelisk-linux-${TARGETARCH} \
 && chmod +x /usr/local/bin/bazelisk \
 && mkdir /build \
 && chown nobody:nogroup /build \
 && usermod --home /build nobody

USER nobody
WORKDIR /build

RUN git clone --depth 1 --branch v1.67.1 https://github.com/grpc/grpc
WORKDIR /build/grpc
RUN bazelisk build //src/compiler:grpc_plugin_support
RUN bazelisk build //src/compiler:grpc_ruby_plugin.stripped

FROM gcr.io/distroless/cc-debian12:latest@sha256:6f05aba4de16e89f8d879bf2a1364de3e41aba04f1dcbba8c75494f6134b4b13 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 /build/grpc/bazel-bin/src/compiler/grpc_ruby_plugin .
USER nobody
ENTRYPOINT ["/grpc_ruby_plugin"]

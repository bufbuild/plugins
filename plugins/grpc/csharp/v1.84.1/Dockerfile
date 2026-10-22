# syntax=docker/dockerfile:1.27
FROM debian:trixie-20261005@sha256:913f6706df59a68922d1dd08f78c2476560a8d367897200a6005b00e5f67c2d5 AS build

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

RUN git clone --depth 1 --branch v1.84.1 https://github.com/grpc/grpc
WORKDIR /build/grpc
RUN bazelisk ${BAZEL_OPTS} build //src/compiler:grpc_plugin_support
RUN bazelisk ${BAZEL_OPTS} build //src/compiler:grpc_csharp_plugin

FROM mcr.microsoft.com/dotnet/sdk:8.0.426-bookworm-slim@sha256:738f72aff955d5c5690e06b6fc8ecd1b90ead564099a257ac528ff720b8da85f AS dotnetrestore
WORKDIR /build
COPY --link ./build.csproj /build/build.csproj
RUN mkdir /nuget && dotnet restore --packages /nuget

FROM gcr.io/distroless/cc-debian13:latest@sha256:b2e43dc72acf58a146ed6feaf350806ae3bf4243eb5c68062d06bc13228ca2bd AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=dotnetrestore /nuget /nuget
COPY --link --from=build --chmod=0755 /build/grpc/bazel-bin/src/compiler/grpc_csharp_plugin_binary .
USER nobody
ENTRYPOINT ["/grpc_csharp_plugin_binary"]

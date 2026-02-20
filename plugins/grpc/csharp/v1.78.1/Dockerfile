# syntax=docker/dockerfile:1.19
FROM debian:bookworm-20260202 AS build

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

RUN git clone --depth 1 --branch v1.78.1 https://github.com/grpc/grpc
WORKDIR /build/grpc
RUN bazelisk ${BAZEL_OPTS} build //src/compiler:grpc_plugin_support
RUN bazelisk ${BAZEL_OPTS} build //src/compiler:grpc_csharp_plugin

FROM mcr.microsoft.com/dotnet/sdk:8.0.418-bookworm-slim@sha256:58359d0b8fe8237be1d63ac335ca378e2857f976c7f92791f7c84b3c117037f5 AS dotnetrestore
WORKDIR /build
COPY --link ./build.csproj /build/build.csproj
RUN mkdir /nuget && dotnet restore --packages /nuget

FROM gcr.io/distroless/cc-debian12:latest@sha256:329e54034ce498f9c6b345044e8f530c6691f99e94a92446f68c0adf9baa8464 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=dotnetrestore /nuget /nuget
COPY --link --from=build --chmod=0755 /build/grpc/bazel-bin/src/compiler/grpc_csharp_plugin_binary .
USER nobody
ENTRYPOINT ["/grpc_csharp_plugin_binary"]

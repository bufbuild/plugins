# syntax=docker/dockerfile:1.12
FROM debian:bookworm-20250407 AS build

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

RUN git clone --depth 1 --branch v1.72.0 https://github.com/grpc/grpc
WORKDIR /build/grpc
RUN bazelisk ${BAZEL_OPTS} build //src/compiler:grpc_plugin_support
RUN bazelisk ${BAZEL_OPTS} build //src/compiler:grpc_csharp_plugin

FROM mcr.microsoft.com/dotnet/sdk:8.0.408-bookworm-slim@sha256:1875fc5f4be6211c22c4353aafd6b13279c0175f277fbfa53d52375d99e8a9b8 AS dotnetrestore
WORKDIR /build
COPY --link ./build.csproj /build/build.csproj
RUN mkdir /nuget && dotnet restore --packages /nuget

FROM gcr.io/distroless/cc-debian12:latest@sha256:c1cbcec08d39c81adbefb80cabc51cba285465866f7b5ab15ddb2fcae51a1aed AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=dotnetrestore /nuget /nuget
COPY --link --from=build --chmod=0755 /build/grpc/bazel-bin/src/compiler/grpc_csharp_plugin .
USER nobody
ENTRYPOINT ["/grpc_csharp_plugin"]

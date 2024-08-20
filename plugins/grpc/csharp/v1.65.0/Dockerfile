# syntax=docker/dockerfile:1.8
FROM debian:bookworm-20240612 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl git cmake build-essential autoconf clang libc++-dev libtool pkg-config unzip zip
RUN arch=${TARGETARCH}; \
    if [ "${arch}" = "amd64" ]; then arch="x86_64"; fi; \
    curl -fsSL -o /usr/local/bin/bazel https://github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-7.2.1-linux-${arch} \
 && chmod +x /usr/local/bin/bazel

WORKDIR /build

RUN git clone --depth 1 --branch v1.65.0 https://github.com/grpc/grpc
WORKDIR /build/grpc
RUN bazel build //src/compiler:grpc_plugin_support
RUN bazel build //src/compiler:grpc_csharp_plugin.stripped

FROM mcr.microsoft.com/dotnet/sdk:8.0.401-bookworm-slim@sha256:8c6beed050a602970c3d275756ed3c19065e42ce6ca0809f5a6fcbf5d36fd305 AS dotnetrestore
WORKDIR /build
COPY --link ./build.csproj /build/build.csproj
RUN mkdir /nuget && dotnet restore --packages /nuget

FROM gcr.io/distroless/cc-debian12:latest@sha256:e1065a1d58800a7294f74e67c32ec4146d09d6cbe471c1fa7ed456b2d2bf06e0 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=dotnetrestore --chmod=0755 /nuget /nuget
COPY --link --from=build --chmod=0755 /build/grpc/bazel-bin/src/compiler/grpc_csharp_plugin .
USER nobody
ENTRYPOINT ["/grpc_csharp_plugin"]

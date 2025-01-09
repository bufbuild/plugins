# syntax=docker/dockerfile:1.12
FROM debian:bookworm-20241223 AS build

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
RUN curl -fsSL -o protoc.tar.gz https://github.com/protocolbuffers/protobuf/releases/download/v29.3/protobuf-29.3.tar.gz \
 && tar --strip-components=1 -zxf protoc.tar.gz \
 && rm protoc.tar.gz
RUN bazelisk ${BAZEL_OPTS} build '//:protoc_lib'
COPY --link BUILD csharp.cc plugins/
RUN bazelisk ${BAZEL_OPTS} build '//plugins:protoc-gen-csharp.stripped'

FROM mcr.microsoft.com/dotnet/sdk:8.0.404-bookworm-slim@sha256:f25e4f51fa06e3b14af1a1135013a3e96055b76caa0e76afc0096d64a77879fd AS dotnetrestore
WORKDIR /build
COPY --link ./build.csproj /build/build.csproj
RUN mkdir /nuget && dotnet restore --packages /nuget

FROM gcr.io/distroless/cc-debian12:latest@sha256:f913198471738d9eedcd00c0ca812bf663e8959eebff3a3cbadb027ed9da0c38 AS base

FROM scratch
COPY --from=base --link / /
COPY --link --from=dotnetrestore /nuget /nuget
COPY --from=build --link --chmod=0755 /build/bazel-bin/plugins/protoc-gen-csharp .
USER nobody
ENTRYPOINT ["/protoc-gen-csharp"]

# syntax=docker/dockerfile:1.22
FROM debian:trixie-20260316@sha256:55a15a112b42be10bfc8092fcc40b6748dc236f7ef46a358d9392b339e9d60e8 AS build

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
RUN curl -fsSL -o protoc.tar.gz https://github.com/protocolbuffers/protobuf/releases/download/v34.1/protobuf-34.1.tar.gz \
 && tar --strip-components=1 -zxf protoc.tar.gz \
 && rm protoc.tar.gz
RUN bazelisk ${BAZEL_OPTS} build '//:protoc_lib'
COPY --link BUILD csharp.cc plugins/
RUN bazelisk ${BAZEL_OPTS} build '//plugins:protoc-gen-csharp.stripped'

FROM mcr.microsoft.com/dotnet/sdk:8.0.419-bookworm-slim@sha256:a9330090b730c2abde8f3f43b3d1e24e4b8cba028de7bab1a7fdcd50cb7a3b7e AS dotnetrestore
WORKDIR /build
COPY --link ./build.csproj /build/build.csproj
RUN mkdir /nuget && dotnet restore --packages /nuget

FROM gcr.io/distroless/cc-debian13:latest@sha256:e1cc90d06703f5dc30ae869fbfce78fce688f21a97efecd226375233a882e62f AS base

FROM scratch
COPY --from=base --link / /
COPY --link --from=dotnetrestore /nuget /nuget
COPY --from=build --link --chmod=0755 /build/bazel-bin/plugins/protoc-gen-csharp .
USER nobody
ENTRYPOINT ["/protoc-gen-csharp"]

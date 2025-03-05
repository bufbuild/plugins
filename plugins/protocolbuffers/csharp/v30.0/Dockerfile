# syntax=docker/dockerfile:1.12
FROM debian:bookworm-20250224 AS build

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
RUN curl -fsSL -o protoc.tar.gz https://github.com/protocolbuffers/protobuf/releases/download/v30.0/protobuf-30.0.tar.gz \
 && tar --strip-components=1 -zxf protoc.tar.gz \
 && rm protoc.tar.gz
RUN bazelisk ${BAZEL_OPTS} build '//:protoc_lib'
COPY --link BUILD csharp.cc plugins/
RUN bazelisk ${BAZEL_OPTS} build '//plugins:protoc-gen-csharp.stripped'

FROM mcr.microsoft.com/dotnet/sdk:8.0.406-bookworm-slim@sha256:483d6f3faa583c93d522c4ca9ee54e08e535cb112dceb252b2fbb7ef94839cc8 AS dotnetrestore
WORKDIR /build
COPY --link ./build.csproj /build/build.csproj
RUN mkdir /nuget && dotnet restore --packages /nuget

FROM gcr.io/distroless/cc-debian12:latest@sha256:b7550f0b15838de14c564337eef2b804ba593ae55d81ca855421bd52f19bb480 AS base

FROM scratch
COPY --from=base --link / /
COPY --link --from=dotnetrestore /nuget /nuget
COPY --from=build --link --chmod=0755 /build/bazel-bin/plugins/protoc-gen-csharp .
USER nobody
ENTRYPOINT ["/protoc-gen-csharp"]

# syntax=docker/dockerfile:1.10
FROM debian:bookworm-20240904 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl git cmake build-essential g++ unzip zip
RUN arch=${TARGETARCH}; \
    if [ "${arch}" = "amd64" ]; then arch="x86_64"; fi; \
    curl -fsSL -o /usr/local/bin/bazel https://github.com/bazelbuild/bazel/releases/download/7.3.1/bazel-7.3.1-linux-${arch} \
 && chmod +x /usr/local/bin/bazel

WORKDIR /build
RUN curl -fsSL -o protoc.tar.gz https://github.com/protocolbuffers/protobuf/releases/download/v28.1/protobuf-28.1.tar.gz \
 && tar --strip-components=1 -zxf protoc.tar.gz \
 && rm protoc.tar.gz
RUN bazel build '//:protoc_lib'
COPY --link BUILD csharp.cc plugins/
RUN bazel build '//plugins:protoc-gen-csharp.stripped'

FROM mcr.microsoft.com/dotnet/sdk:8.0.401-1-bookworm-slim@sha256:a364676fedc145cf88caad4bfb3cc372aae41e596c54e8a63900a2a1c8e364c6 AS dotnetrestore
WORKDIR /build
COPY --link ./build.csproj /build/build.csproj
RUN mkdir /nuget && dotnet restore --packages /nuget

FROM gcr.io/distroless/cc-debian12:latest@sha256:b6e1e913f633495eeb80a41e03de1a41aa863e9b19902309b180ffdc4b99db2c AS base

FROM scratch
COPY --from=base --link / /
COPY --link --from=dotnetrestore /nuget /nuget
COPY --from=build --link --chmod=0755 /build/bazel-bin/plugins/protoc-gen-csharp .
USER nobody
ENTRYPOINT ["/protoc-gen-csharp"]

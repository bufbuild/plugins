# syntax=docker/dockerfile:1.10
FROM debian:bookworm-20241016 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl git cmake build-essential g++ unzip zip
RUN arch=${TARGETARCH}; \
    if [ "${arch}" = "amd64" ]; then arch="x86_64"; fi; \
    curl -fsSL -o /usr/local/bin/bazel https://github.com/bazelbuild/bazel/releases/download/7.4.0/bazel-7.4.0-linux-${arch} \
 && chmod +x /usr/local/bin/bazel

WORKDIR /build
RUN curl -fsSL -o protoc.tar.gz https://github.com/protocolbuffers/protobuf/releases/download/v28.3/protobuf-28.3.tar.gz \
 && tar --strip-components=1 -zxf protoc.tar.gz \
 && rm protoc.tar.gz
RUN bazel build '//:protoc_lib'
COPY --link BUILD java.cc plugins/
RUN bazel build '//plugins:protoc-gen-java.stripped'

FROM gcr.io/distroless/cc-debian12:latest@sha256:3310655aac0d85eb9d579792387af1ff3eb7a1667823478be58020ab0e0d97a8 AS base

FROM scratch
COPY --from=base --link / /
COPY --from=build --link --chmod=0755 /build/bazel-bin/plugins/protoc-gen-java .
USER nobody
ENTRYPOINT ["/protoc-gen-java"]

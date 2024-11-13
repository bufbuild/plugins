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
RUN curl -fsSL -o protoc.tar.gz https://github.com/protocolbuffers/protobuf/releases/download/v28.2/protobuf-28.2.tar.gz \
 && tar --strip-components=1 -zxf protoc.tar.gz \
 && rm protoc.tar.gz
RUN bazel build '//:protoc_lib'
COPY --link BUILD objectivec.cc plugins/
RUN bazel build '//plugins:protoc-gen-objectivec.stripped'

FROM gcr.io/distroless/cc-debian12:latest@sha256:b6e1e913f633495eeb80a41e03de1a41aa863e9b19902309b180ffdc4b99db2c AS base

FROM scratch
COPY --from=base --link / /
COPY --from=build --link --chmod=0755 /build/bazel-bin/plugins/protoc-gen-objectivec .
USER nobody
ENTRYPOINT ["/protoc-gen-objectivec"]

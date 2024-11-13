# syntax=docker/dockerfile:1.10
FROM debian:bookworm-20240926 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl git cmake build-essential autoconf clang libc++-dev libtool pkg-config unzip zip
RUN arch=${TARGETARCH}; \
    if [ "${arch}" = "amd64" ]; then arch="x86_64"; fi; \
    curl -fsSL -o /usr/local/bin/bazel https://github.com/bazelbuild/bazel/releases/download/7.3.2/bazel-7.3.2-linux-${arch} \
 && chmod +x /usr/local/bin/bazel

WORKDIR /build

RUN git clone --depth 1 --branch v1.66.2 https://github.com/grpc/grpc
WORKDIR /build/grpc
RUN bazel build //src/compiler:grpc_plugin_support
RUN bazel build //src/compiler:grpc_python_plugin.stripped

FROM gcr.io/distroless/cc-debian12:latest@sha256:3310655aac0d85eb9d579792387af1ff3eb7a1667823478be58020ab0e0d97a8 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 /build/grpc/bazel-bin/src/compiler/grpc_python_plugin .
USER nobody
ENTRYPOINT ["/grpc_python_plugin"]

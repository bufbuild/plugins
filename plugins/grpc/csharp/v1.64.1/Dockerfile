# syntax=docker/dockerfile:1.7
FROM debian:bookworm-20240513 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl git cmake build-essential autoconf clang libc++-dev libtool pkg-config unzip zip
RUN arch=${TARGETARCH}; \
    if [ "${arch}" = "amd64" ]; then arch="x86_64"; fi; \
    curl -fsSL -o /usr/local/bin/bazel https://github.com/bazelbuild/bazel/releases/download/7.1.2/bazel-7.1.2-linux-${arch} \
 && chmod +x /usr/local/bin/bazel

WORKDIR /build

RUN git clone --depth 1 --branch v1.64.1 https://github.com/grpc/grpc
WORKDIR /build/grpc
RUN bazel build //src/compiler:grpc_plugin_support
RUN bazel build //src/compiler:grpc_csharp_plugin.stripped

FROM gcr.io/distroless/cc-debian12:latest@sha256:e1065a1d58800a7294f74e67c32ec4146d09d6cbe471c1fa7ed456b2d2bf06e0
COPY --from=build --link --chmod=0755 /build/grpc/bazel-bin/src/compiler/grpc_csharp_plugin .
USER nobody
ENTRYPOINT ["/grpc_csharp_plugin"]

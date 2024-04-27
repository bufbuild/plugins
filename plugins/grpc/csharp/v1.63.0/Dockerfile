# syntax=docker/dockerfile:1.4
FROM debian:bookworm-20240423 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl git cmake build-essential autoconf clang libc++-dev libtool pkg-config unzip zip
RUN arch=${TARGETARCH}; \
    if [ "${arch}" = "amd64" ]; then arch="x86_64"; fi; \
    curl -fsSL -o /usr/local/bin/bazel https://github.com/bazelbuild/bazel/releases/download/7.1.1/bazel-7.1.1-linux-${arch} \
 && chmod +x /usr/local/bin/bazel

WORKDIR /build

RUN git clone --depth 1 --branch v1.63.0 https://github.com/grpc/grpc
WORKDIR /build/grpc
RUN bazel build //src/compiler:grpc_plugin_support
RUN bazel build //src/compiler:grpc_csharp_plugin.stripped

FROM gcr.io/distroless/cc-debian12:latest@sha256:7a01d633f75120af59c71489e0911fa8b6512673a3ff0b999522b4221ab4d86a
COPY --from=build --link --chmod=0755 /build/grpc/bazel-bin/src/compiler/grpc_csharp_plugin .
USER nobody
ENTRYPOINT ["/grpc_csharp_plugin"]

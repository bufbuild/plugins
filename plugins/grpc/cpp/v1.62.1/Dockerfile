# syntax=docker/dockerfile:1.4
FROM debian:bookworm-20240211 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl git cmake build-essential autoconf clang libc++-dev libtool pkg-config unzip zip
RUN arch=${TARGETARCH}; \
    if [ "${arch}" = "amd64" ]; then arch="x86_64"; fi; \
    curl -fsSL -o /usr/local/bin/bazel https://github.com/bazelbuild/bazel/releases/download/7.0.2/bazel-7.0.2-linux-${arch} \
 && chmod +x /usr/local/bin/bazel

WORKDIR /build

RUN git clone --depth 1 --branch v1.62.1 https://github.com/grpc/grpc
WORKDIR /build/grpc
RUN bazel build //src/compiler:grpc_plugin_support
RUN bazel build //src/compiler:grpc_cpp_plugin.stripped

FROM gcr.io/distroless/cc-debian12:latest@sha256:efafe74d452c57025616c816b058e3d453c184e4b337897a8d38fef5026b079d
COPY --from=build --link --chmod=0755 /build/grpc/bazel-bin/src/compiler/grpc_cpp_plugin .
USER nobody
ENTRYPOINT ["/grpc_cpp_plugin"]

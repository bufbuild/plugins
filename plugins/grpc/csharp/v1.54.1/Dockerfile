# syntax=docker/dockerfile:1.4
FROM debian:bullseye-20230411 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl git cmake build-essential autoconf clang libc++-dev libtool pkg-config g++ unzip zip
RUN arch=${TARGETARCH}; \
    if [ "${arch}" = "amd64" ]; then arch="x86_64"; fi; \
    curl -fsSL -o /usr/local/bin/bazel https://github.com/bazelbuild/bazel/releases/download/6.1.1/bazel-6.1.1-linux-${arch} \
 && chmod +x /usr/local/bin/bazel

WORKDIR /build

RUN git clone --depth 1 --branch v1.54.1 --recursive https://github.com/grpc/grpc
WORKDIR /build/grpc
RUN bazel build //src/compiler:grpc_plugin_support
RUN bazel build //src/compiler:grpc_csharp_plugin.stripped

FROM gcr.io/distroless/cc-debian11
COPY --from=build --link --chmod=0755 /build/grpc/bazel-bin/src/compiler/grpc_csharp_plugin .
USER nobody
ENTRYPOINT ["/grpc_csharp_plugin"]

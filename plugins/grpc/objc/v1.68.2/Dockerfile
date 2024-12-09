# syntax=docker/dockerfile:1.12
FROM debian:bookworm-20241202 AS build

ARG TARGETARCH
ARG BAZEL_OPTS="--host_jvm_args=-Djava.net.preferIPv4Stack=true"

RUN apt-get update \
 && apt-get install -y curl git cmake build-essential autoconf clang libc++-dev libtool pkg-config unzip zip
RUN curl -fsSL -o /usr/local/bin/bazelisk https://github.com/bazelbuild/bazelisk/releases/download/v1.24.1/bazelisk-linux-${TARGETARCH} \
 && chmod +x /usr/local/bin/bazelisk \
 && mkdir /build \
 && chown nobody:nogroup /build \
 && usermod --home /build nobody

USER nobody
WORKDIR /build

RUN git clone --depth 1 --branch v1.68.2 https://github.com/grpc/grpc
WORKDIR /build/grpc
RUN bazelisk ${BAZEL_OPTS} build //src/compiler:grpc_plugin_support
RUN bazelisk ${BAZEL_OPTS} build //src/compiler:grpc_objective_c_plugin.stripped

FROM gcr.io/distroless/cc-debian12:latest@sha256:f913198471738d9eedcd00c0ca812bf663e8959eebff3a3cbadb027ed9da0c38 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 /build/grpc/bazel-bin/src/compiler/grpc_objective_c_plugin .
USER nobody
ENTRYPOINT ["/grpc_objective_c_plugin"]

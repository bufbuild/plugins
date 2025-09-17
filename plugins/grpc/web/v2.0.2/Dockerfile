# syntax=docker/dockerfile:1.18
FROM debian:bookworm-20250908 AS build

ARG TARGETARCH
ARG BAZEL_OPTS="--host_jvm_args=-Djava.net.preferIPv4Stack=true"

RUN apt-get update \
 && apt-get install -y curl git cmake build-essential autoconf clang libc++-dev libtool pkg-config unzip zip
RUN curl -fsSL -o /usr/local/bin/bazelisk https://github.com/bazelbuild/bazelisk/releases/download/v1.27.0/bazelisk-linux-${TARGETARCH} \
 && chmod +x /usr/local/bin/bazelisk \
 && mkdir /build \
 && chown nobody:nogroup /build \
 && usermod --home /build nobody

USER nobody
WORKDIR /build
RUN curl -fsSL -o grpc-web-src.tar.gz https://github.com/grpc/grpc-web/releases/download/2.0.2/grpc-web-source-2.0.2.tar.gz \
 && tar -zxf grpc-web-src.tar.gz \
 && rm grpc-web-src.tar.gz
WORKDIR /build/grpc-web-2.0.2
RUN bazelisk ${BAZEL_OPTS} build '//javascript/net/grpc/web/generator:protoc-gen-grpc-web'

FROM gcr.io/distroless/cc-debian12:latest@sha256:620d8b11ae800f0dbd7995f89ddc5344ad603269ea98770588b1b07a4a0a6872 AS base

FROM scratch
COPY --from=base --link / /
COPY --from=build --link --chmod=0755 /build/grpc-web-2.0.2/bazel-bin/javascript/net/grpc/web/generator/protoc-gen-grpc-web .
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-web" ]

# syntax=docker/dockerfile:1.7
FROM debian:bookworm-20240513 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl git cmake build-essential g++ unzip zip
RUN arch=${TARGETARCH}; \
    if [ "${arch}" = "amd64" ]; then arch="x86_64"; fi; \
    curl -fsSL -o /usr/local/bin/bazel https://github.com/bazelbuild/bazel/releases/download/7.1.2/bazel-7.1.2-linux-${arch} \
 && chmod +x /usr/local/bin/bazel

WORKDIR /build
RUN curl -fsSL -o protoc.tar.gz https://github.com/protocolbuffers/protobuf/releases/download/v27.0/protobuf-27.0.tar.gz \
 && tar --strip-components=1 -zxf protoc.tar.gz \
 && rm protoc.tar.gz
RUN bazel build '//:protoc_lib'
COPY --link BUILD ruby.cc plugins/
RUN bazel build '//plugins:protoc-gen-ruby.stripped'

FROM gcr.io/distroless/cc-debian12:latest@sha256:e1065a1d58800a7294f74e67c32ec4146d09d6cbe471c1fa7ed456b2d2bf06e0
COPY --from=build --link --chmod=0755 /build/bazel-bin/plugins/protoc-gen-ruby .
USER nobody
ENTRYPOINT ["/protoc-gen-ruby"]

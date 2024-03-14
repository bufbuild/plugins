# syntax=docker/dockerfile:1.5
FROM debian:bookworm-20240311 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl git cmake build-essential g++ unzip zip
RUN arch=${TARGETARCH}; \
    if [ "${arch}" = "amd64" ]; then arch="x86_64"; fi; \
    curl -fsSL -o /usr/local/bin/bazel https://github.com/bazelbuild/bazel/releases/download/7.0.2/bazel-7.0.2-linux-${arch} \
 && chmod +x /usr/local/bin/bazel

WORKDIR /build
RUN curl -fsSL -o protoc.tar.gz https://github.com/protocolbuffers/protobuf/releases/download/v26.0/protobuf-26.0.tar.gz \
 && tar --strip-components=1 -zxf protoc.tar.gz \
 && rm protoc.tar.gz
RUN bazel build '//:protoc_lib'
COPY --link BUILD java.cc plugins/
RUN bazel build '//plugins:protoc-gen-java.stripped'

FROM gcr.io/distroless/cc-debian12:latest@sha256:e6ae66a5a343d7112167f9117c4e630cfffcd80db44e44302759ec13ddd2d22b
COPY --from=build --link --chmod=0755 /build/bazel-bin/plugins/protoc-gen-java .
USER nobody
ENTRYPOINT ["/protoc-gen-java"]

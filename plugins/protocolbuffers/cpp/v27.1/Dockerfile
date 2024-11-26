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
RUN curl -fsSL -o protoc.tar.gz https://github.com/protocolbuffers/protobuf/releases/download/v27.1/protobuf-27.1.tar.gz \
 && tar --strip-components=1 -zxf protoc.tar.gz \
 && rm protoc.tar.gz
RUN bazel build '//:protoc_lib'
COPY --link BUILD cpp.cc plugins/
RUN bazel build '//plugins:protoc-gen-cpp.stripped'

FROM gcr.io/distroless/cc-debian12:latest@sha256:2fb69596e692931f909c4c69ab09e50608959eaf8898c44fa64db741a23588b0 AS base

FROM scratch
COPY --from=base --link / /
COPY --from=build --link --chmod=0755 /build/bazel-bin/plugins/protoc-gen-cpp .
USER nobody
ENTRYPOINT ["/protoc-gen-cpp"]

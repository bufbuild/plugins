# syntax=docker/dockerfile:1.4
FROM debian:bullseye-20221004 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl git cmake build-essential g++ unzip zip
RUN arch=${TARGETARCH}; \
    if [ "${arch}" = "amd64" ]; then arch="x86_64"; fi; \
    curl -fsSL -o /usr/local/bin/bazel https://github.com/bazelbuild/bazel/releases/download/5.3.1/bazel-5.3.1-linux-${arch} \
 && chmod +x /usr/local/bin/bazel

WORKDIR /build
RUN git clone https://github.com/protocolbuffers/protobuf --depth 1 --branch v21.7 --recursive
WORKDIR /build/protobuf/
RUN bazel build '//:protoc_lib'
COPY BUILD ruby.cc plugins/ruby/
RUN bazel build '//plugins/ruby:protoc-gen-ruby'

FROM debian:bullseye-20221004-slim
COPY --from=build --link --chmod=0755 /build/protobuf/bazel-bin/plugins/ruby/protoc-gen-ruby .
USER nobody
ENTRYPOINT ["/protoc-gen-ruby"]

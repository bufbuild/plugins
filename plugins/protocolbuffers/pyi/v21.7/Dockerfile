# syntax=docker/dockerfile:1.4
FROM debian:bullseye-20221024 AS build

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
COPY --link BUILD pyi.cc plugins/
RUN bazel build '//plugins:protoc-gen-pyi'

FROM debian:bullseye-20221024-slim
COPY --from=build --link --chmod=0755 /build/protobuf/bazel-bin/plugins/protoc-gen-pyi .
USER nobody
ENTRYPOINT ["/protoc-gen-pyi"]

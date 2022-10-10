# syntax=docker/dockerfile:1.4
FROM debian:bullseye-20220822 AS build

ARG TARGETARCH
ARG BAZEL_VERSION="4.2.1"

RUN apt-get update \
 && apt-get install -y curl git cmake build-essential g++ unzip zip
RUN arch=${TARGETARCH}; \
    if [ "${arch}" = "amd64" ]; then arch="x86_64"; fi; \
    curl -fsSL -o /usr/local/bin/bazel https://github.com/bazelbuild/bazel/releases/download/${BAZEL_VERSION}/bazel-${BAZEL_VERSION}-linux-${arch} \
 && chmod +x /usr/local/bin/bazel

WORKDIR /build
RUN git clone https://github.com/protocolbuffers/protobuf-javascript --depth 1 --branch v3.21.1 --recursive
WORKDIR /build/protobuf-javascript
RUN bazel build //generator:protoc-gen-js

FROM debian:bullseye-20220822-slim
COPY --from=build --link --chmod=0755 /build/protobuf-javascript/bazel-bin/generator/protoc-gen-js .
USER nobody
ENTRYPOINT ["/protoc-gen-js"]

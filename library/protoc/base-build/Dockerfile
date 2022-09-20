# syntax=docker/dockerfile:1.4
FROM debian:bullseye-20220912

ARG TARGETARCH
ARG BAZEL_VERSION="4.2.1"

RUN apt-get update \
 && apt-get install -y curl git cmake build-essential g++ unzip zip
RUN arch=${TARGETARCH}; \
    if [ "${arch}" = "amd64" ]; then arch="x86_64"; fi; \
    curl -fsSL -o /usr/local/bin/bazel https://github.com/bazelbuild/bazel/releases/download/${BAZEL_VERSION}/bazel-${BAZEL_VERSION}-linux-${arch} \
 && chmod +x /usr/local/bin/bazel

WORKDIR /build
COPY --link plugins plugins

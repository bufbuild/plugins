# syntax=docker/dockerfile:1.5
FROM debian:bullseye-20230703 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl git cmake build-essential g++ unzip zip
RUN arch=${TARGETARCH}; \
    if [ "${arch}" = "amd64" ]; then arch="x86_64"; fi; \
    curl -fsSL -o /usr/local/bin/bazel https://github.com/bazelbuild/bazel/releases/download/6.2.1/bazel-6.2.1-linux-${arch} \
 && chmod +x /usr/local/bin/bazel

WORKDIR /build
RUN curl -fsSL -o protoc.tar.gz https://github.com/protocolbuffers/protobuf/releases/download/v23.4/protobuf-23.4.tar.gz \
 && tar --strip-components=1 -zxf protoc.tar.gz \
 && rm protoc.tar.gz
COPY escape-package-names.patch .
RUN patch -p1 -u < escape-package-names.patch
RUN bazel build '//:protoc_lib'
COPY --link BUILD kotlin.cc plugins/
RUN bazel build '//plugins:protoc-gen-kotlin.stripped'

FROM gcr.io/distroless/cc-debian11
COPY --from=build --link --chmod=0755 /build/bazel-bin/plugins/protoc-gen-kotlin .
USER nobody
ENTRYPOINT ["/protoc-gen-kotlin"]

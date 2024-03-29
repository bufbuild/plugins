# syntax=docker/dockerfile:1.4
FROM debian:bullseye-20230502 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl git cmake build-essential g++ unzip zip
RUN arch=${TARGETARCH}; \
    if [ "${arch}" = "amd64" ]; then arch="x86_64"; fi; \
    curl -fsSL -o /usr/local/bin/bazel https://github.com/bazelbuild/bazel/releases/download/6.1.2/bazel-6.1.2-linux-${arch} \
 && chmod +x /usr/local/bin/bazel

WORKDIR /build
RUN curl -fsSL -o protoc.tar.gz https://github.com/protocolbuffers/protobuf/releases/download/v22.4/protobuf-22.4.tar.gz \
 && tar --strip-components=1 -zxf protoc.tar.gz \
 && rm protoc.tar.gz
RUN bazel build '//:protoc_lib'
RUN mkdir -p plugins/pyi/
COPY <<'EOF' /build/plugins/pyi/pyi.cc
#include <google/protobuf/compiler/python/pyi_generator.h>
#include <google/protobuf/compiler/plugin.h>

int main(int argc, char *argv[]) {
  google::protobuf::compiler::python::PyiGenerator generator;
  return google::protobuf::compiler::PluginMain(argc, argv, &generator);
}
EOF
COPY <<'EOF' /build/plugins/pyi/BUILD
cc_binary(
    name = "protoc-gen-pyi",
    srcs = ["pyi.cc"],
    deps = [
        "//:protoc_lib",
    ],
)
EOF
RUN bazel build '//plugins/pyi:protoc-gen-pyi.stripped'

FROM gcr.io/distroless/cc-debian11
COPY --from=build --link --chmod=0755 /build/bazel-bin/plugins/pyi/protoc-gen-pyi .
USER nobody
ENTRYPOINT ["/protoc-gen-pyi"]

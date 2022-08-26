# syntax=docker/dockerfile:1.4
FROM debian:bullseye-20220822 AS build

ARG PLUGIN_VERSION
RUN test -n "${PLUGIN_VERSION}"

ARG TARGETARCH
ARG BAZEL_VERSION="4.2.1"

WORKDIR /build
RUN /bin/bash <<'EOF'
set -e
if [ "${TARGETARCH}" = "arm64" ]; then
    apt-get update
    apt-get install -y curl git build-essential g++
    curl -fsSL -o /usr/local/bin/bazel https://github.com/bazelbuild/bazel/releases/download/${BAZEL_VERSION}/bazel-${BAZEL_VERSION}-linux-${TARGETARCH}
    chmod +x /usr/local/bin/bazel
    git clone --recursive --depth 1 --branch ${PLUGIN_VERSION#v} https://github.com/grpc/grpc-web.git
    cd grpc-web
    bazel build javascript/net/grpc/web/generator:protoc-gen-grpc-web
    mv bazel-bin/javascript/net/grpc/web/generator/protoc-gen-grpc-web /build/protoc-gen-grpc-web
else
    apt-get update
    apt-get install -y curl
    curl -fsSL https://github.com/grpc/grpc-web/releases/download/${PLUGIN_VERSION#v}/protoc-gen-grpc-web-${PLUGIN_VERSION#v}-linux-x86_64 -o protoc-gen-grpc-web
fi
EOF

FROM debian:bullseye-20220822-slim
COPY --from=build --link --chmod=0755 --chown=root:root /build/protoc-gen-grpc-web .
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-web" ]

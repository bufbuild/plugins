# syntax=docker/dockerfile:1.4
FROM node:16.17.0-bullseye AS build

ARG PLUGIN_VERSION
RUN test -n "${PLUGIN_VERSION}"

ARG TARGETARCH

WORKDIR /build
COPY --link package* .
RUN /bin/bash <<'EOF'
set -e
# https://github.com/grpc/grpc-node/issues/1405#issuecomment-1195903603 - linux/arm64 binaries not available
if [ "${TARGETARCH}" = "arm64" ]; then
    apt update
    apt install -y git build-essential file jq cmake
    git clone --recursive --depth 1 --branch grpc-tools@${PLUGIN_VERSION#v} https://github.com/grpc/grpc-node.git
    cd grpc-node/packages/grpc-tools
    sed -i -e '# -m64##g' linux_64bit.toolchain.cmake
    cmake linux_64bit.toolchain.cmake . && cmake --build . --target clean && cmake --build . -- -j 12
    cp grpc_node_plugin /build
else
    npm ci
    cp /build/node_modules/grpc-tools/bin/grpc_node_plugin /build
fi
EOF

FROM debian:bullseye-20220822-slim
COPY --from=build --link --chmod=0755 --chown=root:root /build/grpc_node_plugin .
USER nobody
ENTRYPOINT [ "/grpc_node_plugin" ]

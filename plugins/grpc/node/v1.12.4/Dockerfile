# syntax=docker/dockerfile:1.4
FROM node:18.13.0-bullseye AS build

ARG TARGETARCH

WORKDIR /build
COPY --link package* .
RUN npm ci \
 && cp /build/node_modules/grpc-tools/bin/grpc_node_plugin /build

FROM gcr.io/distroless/cc-debian11
COPY --from=build --link --chmod=0755 --chown=root:root /build/grpc_node_plugin .
USER nobody
ENTRYPOINT [ "/grpc_node_plugin" ]

# syntax=docker/dockerfile:1.12
FROM node:22.13.1-bookworm AS build

ARG TARGETARCH

WORKDIR /build
COPY --link package* .
RUN npm ci \
 && cp /build/node_modules/grpc-tools/bin/grpc_node_plugin /build

FROM gcr.io/distroless/cc-debian12:latest@sha256:b7550f0b15838de14c564337eef2b804ba593ae55d81ca855421bd52f19bb480
COPY --from=build --link --chmod=0755 --chown=root:root /build/grpc_node_plugin .
USER nobody
ENTRYPOINT [ "/grpc_node_plugin" ]

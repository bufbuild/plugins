# syntax=docker/dockerfile:1.7
FROM node:20.12.1-alpine3.19 AS build

ARG TARGETARCH

WORKDIR /build
COPY --link package* .
RUN npm ci \
 && cp /build/node_modules/grpc-tools/bin/grpc_node_plugin /build

FROM gcr.io/distroless/cc-debian12:latest@sha256:7a01d633f75120af59c71489e0911fa8b6512673a3ff0b999522b4221ab4d86a
COPY --from=build --link --chmod=0755 --chown=root:root /build/grpc_node_plugin .
USER nobody
ENTRYPOINT [ "/grpc_node_plugin" ]

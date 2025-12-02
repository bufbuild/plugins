# syntax=docker/dockerfile:1.19
FROM node:24.11.1-bookworm AS build

ARG TARGETARCH

WORKDIR /build
COPY --link package* .
RUN npm ci \
 && cp /build/node_modules/grpc-tools/bin/grpc_node_plugin /build

FROM gcr.io/distroless/cc-debian12:latest@sha256:0000f9dc0290f8eaf0ecceafbc35e803649087ea7879570fbc78372df7ac649b AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 /build/grpc_node_plugin .
USER nobody
ENTRYPOINT [ "/grpc_node_plugin" ]

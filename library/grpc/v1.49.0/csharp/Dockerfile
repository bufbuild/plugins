# syntax=docker/dockerfile:1.4
ARG PLUGIN_VERSION
ARG DOCKER_ORG=bufbuild
FROM ${DOCKER_ORG}/plugins-grpc-base:${PLUGIN_VERSION} AS build

FROM debian:bullseye-20220822-slim
COPY --from=build --link --chmod=0755 /build/grpc/grpc_csharp_plugin .
USER nobody
ENTRYPOINT ["/grpc_csharp_plugin"]

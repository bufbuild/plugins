# syntax=docker/dockerfile:1.4
ARG PLUGIN_VERSION
ARG DOCKER_ORG=bufbuild
FROM ${DOCKER_ORG}/plugins-grpc-base:${PLUGIN_VERSION} as build

FROM debian:bullseye-20220822-slim
WORKDIR /app
COPY --from=build --chmod=0755 /build/grpc/grpc_python_plugin .
USER nobody
ENTRYPOINT ["/app/grpc_python_plugin"]

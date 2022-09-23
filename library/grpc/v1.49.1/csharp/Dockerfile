# syntax=docker/dockerfile:1.4
ARG DOCKER_ORG=bufbuild
FROM ${DOCKER_ORG}/plugins-grpc-base:v1.49.1 AS build

FROM debian:bullseye-20220912-slim
COPY --from=build --link --chmod=0755 /build/grpc/grpc_csharp_plugin .
USER nobody
ENTRYPOINT ["/grpc_csharp_plugin"]

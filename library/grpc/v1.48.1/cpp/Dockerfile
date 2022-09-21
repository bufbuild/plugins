# syntax=docker/dockerfile:1.4
ARG DOCKER_ORG=bufbuild
FROM ${DOCKER_ORG}/plugins-grpc-base:v1.48.1 as build

FROM debian:bullseye-20220822-slim
COPY --from=build --link --chmod=0755 /build/grpc/grpc_cpp_plugin .
USER nobody
ENTRYPOINT ["/grpc_cpp_plugin"]

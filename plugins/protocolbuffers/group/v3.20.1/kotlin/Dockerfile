# syntax=docker/dockerfile:1.4
ARG DOCKER_ORG=bufbuild
FROM ${DOCKER_ORG}/plugins-protoc-base:v3.20.1 as build

FROM debian:bullseye-20220822-slim
COPY --from=build --link --chmod=0755 /build/protobuf/bazel-bin/plugins/protoc-gen-kotlin .
USER nobody
ENTRYPOINT ["/protoc-gen-kotlin"]

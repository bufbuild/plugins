# syntax=docker/dockerfile:1.4
ARG DOCKER_ORG=bufbuild
FROM ${DOCKER_ORG}/plugins-protoc-base:v21.7 AS build

FROM debian:bullseye-20220912-slim
COPY --from=build --link --chmod=0755 /build/protobuf/bazel-bin/plugins/protoc-gen-objectivec .
USER nobody
ENTRYPOINT ["/protoc-gen-objectivec"]

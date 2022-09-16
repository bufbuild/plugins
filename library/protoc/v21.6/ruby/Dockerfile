# syntax=docker/dockerfile:1.4
ARG PLUGIN_VERSION
ARG DOCKER_ORG=bufbuild
FROM ${DOCKER_ORG}/plugins-protoc-base:${PLUGIN_VERSION} as build

FROM debian:bullseye-20220822-slim
COPY --from=build --link --chmod=0755 /build/protobuf/bazel-bin/plugins/protoc-gen-ruby .
USER nobody
ENTRYPOINT ["/protoc-gen-ruby"]

# syntax=docker/dockerfile:1.4
ARG PLUGIN_VERSION
FROM bufbuild/plugins-protoc-base:${PLUGIN_VERSION} as build

FROM debian:bullseye-slim
WORKDIR /app
COPY --from=build --chmod=0755 /build/protobuf/bazel-bin/plugins/protoc-gen-ruby .
USER nobody
ENTRYPOINT ["/app/protoc-gen-ruby"]

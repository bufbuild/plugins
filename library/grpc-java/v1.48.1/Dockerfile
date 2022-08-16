# syntax=docker/dockerfile:1.4
FROM debian:bullseye-slim AS build

ARG PLUGIN_VERSION
RUN test -n "${PLUGIN_VERSION}"

ARG TARGETARCH

WORKDIR /build
RUN apt-get update \
 && apt-get install -y curl
RUN arch=${TARGETARCH}; \
    if [ "${arch}" = "arm64" ]; then\
        arch="aarch_64"; \
    elif [ "${arch}" = "amd64" ]; then\
        arch="x86_64"; \
    fi; \
    echo "${arch}"; \
    curl -fsSL -o protoc-gen-grpc-java https://repo1.maven.org/maven2/io/grpc/protoc-gen-grpc-java/${PLUGIN_VERSION#v}/protoc-gen-grpc-java-${PLUGIN_VERSION#v}-linux-${arch}.exe

FROM debian:bullseye-slim
WORKDIR /app
COPY --from=build --chmod=0755 --chown=root:root /build/protoc-gen-grpc-java .
USER nobody
ENTRYPOINT [ "/app/protoc-gen-grpc-java" ]

# syntax=docker/dockerfile:1.10
FROM debian:bookworm-20241016 AS build

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
    curl -fsSL -o protoc-gen-grpc-java https://repo1.maven.org/maven2/io/grpc/protoc-gen-grpc-java/1.68.1/protoc-gen-grpc-java-1.68.1-linux-${arch}.exe

FROM gcr.io/distroless/cc-debian12:latest@sha256:6f05aba4de16e89f8d879bf2a1364de3e41aba04f1dcbba8c75494f6134b4b13 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 --chown=root:root /build/protoc-gen-grpc-java .
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-java" ]

# syntax=docker/dockerfile:1.19
FROM debian:bookworm-20251208 AS build

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
    curl -fsSL -o protoc-gen-grpc-java https://repo1.maven.org/maven2/io/grpc/protoc-gen-grpc-java/1.78.0/protoc-gen-grpc-java-1.78.0-linux-${arch}.exe

FROM gcr.io/distroless/cc-debian12:latest@sha256:0c8eac8ea42a167255d03c3ba6dfad2989c15427ed93d16c53ef9706ea4691df AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 --chown=root:root /build/protoc-gen-grpc-java .
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-java" ]

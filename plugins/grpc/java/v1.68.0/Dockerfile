# syntax=docker/dockerfile:1.10
FROM debian:bookworm-20240904 AS build

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
    curl -fsSL -o protoc-gen-grpc-java https://repo1.maven.org/maven2/io/grpc/protoc-gen-grpc-java/1.68.0/protoc-gen-grpc-java-1.68.0-linux-${arch}.exe

FROM gcr.io/distroless/cc-debian12:latest@sha256:682ff941956437ab1fc0f6fe969b18ede078839cc4f4fbc156ab546d2a9055fd AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 --chown=root:root /build/protoc-gen-grpc-java .
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-java" ]

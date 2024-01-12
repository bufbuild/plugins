# syntax=docker/dockerfile:1.4
FROM debian:bookworm-20240110 AS build

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
    curl -fsSL -o protoc-gen-grpc-java https://repo1.maven.org/maven2/io/grpc/protoc-gen-grpc-java/1.61.0/protoc-gen-grpc-java-1.61.0-linux-${arch}.exe

FROM gcr.io/distroless/base-debian12:latest@sha256:0a93daa199e7c6e387cea8cf03fac676146735caf6965d276d86ebd3a441f27e
COPY --from=build --link --chmod=0755 --chown=root:root /build/protoc-gen-grpc-java .
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-java" ]

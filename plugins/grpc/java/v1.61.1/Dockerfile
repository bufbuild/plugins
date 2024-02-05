# syntax=docker/dockerfile:1.4
FROM debian:bookworm-20240130 AS build

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
    curl -fsSL -o protoc-gen-grpc-java https://repo1.maven.org/maven2/io/grpc/protoc-gen-grpc-java/1.61.1/protoc-gen-grpc-java-1.61.1-linux-${arch}.exe

FROM gcr.io/distroless/base-debian12:latest@sha256:f47fa3dbb9c1b1a5d968106c98380c40f28c721f0f8e598e8d760169ae2db836
COPY --from=build --link --chmod=0755 --chown=root:root /build/protoc-gen-grpc-java .
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-java" ]

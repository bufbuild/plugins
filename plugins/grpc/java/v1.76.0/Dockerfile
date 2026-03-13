# syntax=docker/dockerfile:1.18
FROM debian:bookworm-20250929 AS build

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
    curl -fsSL -o protoc-gen-grpc-java https://repo1.maven.org/maven2/io/grpc/protoc-gen-grpc-java/1.76.0/protoc-gen-grpc-java-1.76.0-linux-${arch}.exe

FROM gcr.io/distroless/cc-debian12:latest@sha256:14f6999db515330e5d00537bd457289a8968b6456e9197c7a28101ee63a7522f AS base

FROM maven:3.9.11-eclipse-temurin-21 AS maven-deps
COPY pom.xml /tmp/pom.xml
RUN cd /tmp && mvn -f pom.xml dependency:copy-dependencies -Dmdep.useRepositoryLayout=true -Dmdep.copyPom=true -DoutputDirectory=/maven-repository && cd /root/.m2/repository && find . -name '*.pom' -exec sh -c 'mkdir -p "/maven-repository/$(dirname "$1")" && cp -n "$1" "/maven-repository/$1"' _ {} \;

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 --chown=root:root /build/protoc-gen-grpc-java .
COPY --from=maven-deps /maven-repository /maven-repository
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-java" ]

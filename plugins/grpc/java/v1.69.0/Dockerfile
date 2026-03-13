# syntax=docker/dockerfile:1.12
FROM debian:bookworm-20241202 AS build

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
    curl -fsSL -o protoc-gen-grpc-java https://repo1.maven.org/maven2/io/grpc/protoc-gen-grpc-java/1.69.0/protoc-gen-grpc-java-1.69.0-linux-${arch}.exe

FROM gcr.io/distroless/cc-debian12:latest@sha256:f913198471738d9eedcd00c0ca812bf663e8959eebff3a3cbadb027ed9da0c38 AS base

FROM maven:3.9.11-eclipse-temurin-21 AS maven-deps
COPY pom.xml /tmp/pom.xml
RUN cd /tmp && mvn -f pom.xml dependency:copy-dependencies -Dmdep.useRepositoryLayout=true -Dmdep.copyPom=true -DoutputDirectory=/maven-repository && cd /root/.m2/repository && find . -name '*.pom' -exec sh -c 'mkdir -p "/maven-repository/$(dirname "$1")" && cp -n "$1" "/maven-repository/$1"' _ {} \;

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 --chown=root:root /build/protoc-gen-grpc-java .
COPY --from=maven-deps /maven-repository /maven-repository
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-java" ]

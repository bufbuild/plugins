# syntax=docker/dockerfile:1.15
FROM debian:bookworm-20250520 AS build

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
    curl -fsSL -o protoc-gen-grpc-java https://repo1.maven.org/maven2/io/grpc/protoc-gen-grpc-java/1.73.0/protoc-gen-grpc-java-1.73.0-linux-${arch}.exe

FROM gcr.io/distroless/cc-debian12:latest@sha256:c53c9416a1acdbfd6e09abba720442444a3d1a6338b8db850e5e198b59af5570 AS base

FROM maven:3.9.11-eclipse-temurin-21 AS maven-deps
COPY pom.xml /tmp/pom.xml
RUN cd /tmp && mvn -f pom.xml dependency:copy-dependencies -Dmdep.useRepositoryLayout=true -Dmdep.copyPom=true -DoutputDirectory=/maven-repository && cd /root/.m2/repository && find . -name '*.pom' -exec cp --parents -t /maven-repository {} +

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 --chown=root:root /build/protoc-gen-grpc-java .
COPY --from=maven-deps /maven-repository /maven-repository
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-java" ]

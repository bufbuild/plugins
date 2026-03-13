# syntax=docker/dockerfile:1.18
FROM debian:bookworm-20250908 AS build

ARG TARGETARCH

WORKDIR /build
RUN apt-get update \
 && apt-get install -y curl
RUN curl -fsSL -o protoc-gen-grpc-kotlin.jar https://repo1.maven.org/maven2/io/grpc/protoc-gen-grpc-kotlin/1.5.0/protoc-gen-grpc-kotlin-1.5.0-jdk8.jar

FROM gcr.io/distroless/java21-debian12:latest@sha256:418b2e2a9e452aa9299511427f2ae404dfc910ecfa78feb53b1c60c22c3b640c AS base

FROM maven:3.9.11-eclipse-temurin-21 AS maven-deps
COPY pom.xml /tmp/pom.xml
RUN cd /tmp && mvn -f pom.xml dependency:copy-dependencies -Dmdep.useRepositoryLayout=true -Dmdep.copyPom=true -DoutputDirectory=/maven-repository && cd /root/.m2/repository && find . -name '*.pom' -exec sh -c 'mkdir -p "/maven-repository/$(dirname "$1")" && cp -n "$1" "/maven-repository/$1"' _ {} \;

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0644 --chown=root:root /build/protoc-gen-grpc-kotlin.jar .
COPY --from=maven-deps /maven-repository /maven-repository
USER nobody
ENTRYPOINT [ "/usr/bin/java", "-jar", "/protoc-gen-grpc-kotlin.jar" ]

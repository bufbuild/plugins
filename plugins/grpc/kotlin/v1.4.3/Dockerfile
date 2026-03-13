# syntax=docker/dockerfile:1.12
FROM debian:bookworm-20250407 AS build

ARG TARGETARCH

WORKDIR /build
RUN apt-get update \
 && apt-get install -y curl
RUN curl -fsSL -o protoc-gen-grpc-kotlin.jar https://repo1.maven.org/maven2/io/grpc/protoc-gen-grpc-kotlin/1.4.3/protoc-gen-grpc-kotlin-1.4.3-jdk8.jar

FROM gcr.io/distroless/java21-debian12:latest@sha256:c298bfc8c8b1aa3d7b03480dcf52001a90d66d966f6a8d8997ae837d3982be3f AS base

FROM maven:3.9.11-eclipse-temurin-21 AS maven-deps
COPY pom.xml /tmp/pom.xml
RUN cd /tmp && mvn -f pom.xml dependency:go-offline

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0644 --chown=root:root /build/protoc-gen-grpc-kotlin.jar .
COPY --from=maven-deps /root/.m2/repository /maven-repository
USER nobody
ENTRYPOINT [ "/usr/bin/java", "-jar", "/protoc-gen-grpc-kotlin.jar" ]

# syntax=docker/dockerfile:1.4
FROM debian:bullseye-20231030 AS build

ARG TARGETARCH

WORKDIR /build
RUN apt-get update \
 && apt-get install -y curl
RUN curl -fsSL -o protoc-gen-grpc-kotlin.jar https://repo1.maven.org/maven2/io/grpc/protoc-gen-grpc-kotlin/1.4.1/protoc-gen-grpc-kotlin-1.4.1-jdk8.jar

FROM maven:3.9.11-eclipse-temurin-21 AS maven-deps
COPY pom.xml /tmp/pom.xml
RUN cd /tmp && mvn -f pom.xml dependency:go-offline

FROM gcr.io/distroless/java11-debian11
COPY --from=build --link --chmod=0644 --chown=root:root /build/protoc-gen-grpc-kotlin.jar .
COPY --from=maven-deps /root/.m2/repository /maven-repository
USER nobody
ENTRYPOINT [ "/usr/bin/java", "-jar", "/protoc-gen-grpc-kotlin.jar" ]

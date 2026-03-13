# syntax=docker/dockerfile:1.8
FROM debian:bookworm-20240701 AS build

RUN apt-get update \
    && apt-get install -y curl
WORKDIR /app
RUN curl -fsSL -o /app/protoc-gen-connect-kotlin.jar https://repo1.maven.org/maven2/com/connectrpc/protoc-gen-connect-kotlin/0.7.0/protoc-gen-connect-kotlin-0.7.0.jar

FROM gcr.io/distroless/java17-debian12:latest@sha256:c8c2c236c63d7652b707a05b02d0259db91d5feb357e9dc53942f73c2e001088 as base

FROM maven:3.9.11-eclipse-temurin-21 AS maven-deps
COPY pom.xml /tmp/pom.xml
RUN cd /tmp && mvn -f pom.xml dependency:go-offline

FROM scratch
COPY --from=base --link / /
COPY --from=build --link --chmod=0755 --chown=root:root /app/protoc-gen-connect-kotlin.jar .
COPY --from=maven-deps /root/.m2/repository /maven-repository
USER nobody
ENTRYPOINT [ "/usr/bin/java", "-jar", "/protoc-gen-connect-kotlin.jar"]

# syntax=docker/dockerfile:1.10
FROM debian:bookworm-20241016 AS build

RUN apt-get update \
    && apt-get install -y curl
WORKDIR /app
RUN curl -fsSL -o /app/protoc-gen-connect-kotlin.jar https://repo1.maven.org/maven2/com/connectrpc/protoc-gen-connect-kotlin/0.7.1/protoc-gen-connect-kotlin-0.7.1.jar

FROM gcr.io/distroless/java17-debian12:latest@sha256:26054428ef0fa1b71d28018e35823060c9e89d4b2f120d8efe1964669f44fccc as base

FROM maven:3.9.11-eclipse-temurin-21 AS maven-deps
COPY pom.xml /tmp/pom.xml
RUN cd /tmp && mvn -f pom.xml dependency:go-offline

FROM scratch
COPY --from=base --link / /
COPY --from=build --link --chmod=0755 --chown=root:root /app/protoc-gen-connect-kotlin.jar .
COPY --from=maven-deps /root/.m2/repository /maven-repository
USER nobody
ENTRYPOINT [ "/usr/bin/java", "-jar", "/protoc-gen-connect-kotlin.jar"]

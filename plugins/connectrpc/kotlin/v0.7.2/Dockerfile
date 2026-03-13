# syntax=docker/dockerfile:1.12
FROM debian:bookworm-20241223 AS build

RUN apt-get update \
    && apt-get install -y curl
WORKDIR /app
RUN curl -fsSL -o /app/protoc-gen-connect-kotlin.jar https://repo1.maven.org/maven2/com/connectrpc/protoc-gen-connect-kotlin/0.7.2/protoc-gen-connect-kotlin-0.7.2.jar

FROM gcr.io/distroless/java21-debian12:latest@sha256:903d5ad227a4afff8a207cd25c580ed059cc4006bb390eae65fb0361fc9724c3 as base

FROM maven:3.9.11-eclipse-temurin-21 AS maven-deps
COPY pom.xml /tmp/pom.xml
RUN cd /tmp && mvn -f pom.xml dependency:go-offline

FROM scratch
COPY --from=base --link / /
COPY --from=build --link --chmod=0755 --chown=root:root /app/protoc-gen-connect-kotlin.jar .
COPY --from=maven-deps /root/.m2/repository /maven-repository
USER nobody
ENTRYPOINT [ "/usr/bin/java", "-jar", "/protoc-gen-connect-kotlin.jar"]

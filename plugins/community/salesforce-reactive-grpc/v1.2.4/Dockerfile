# syntax=docker/dockerfile:1.15
FROM debian:bookworm-20250610 AS build

WORKDIR /app
RUN apt-get update && apt-get install -y curl
RUN curl -fsSL -o reactor-grpc-protoc.jar https://repo1.maven.org/maven2/com/salesforce/servicelibs/reactor-grpc/1.2.4/reactor-grpc-1.2.4.jar

FROM gcr.io/distroless/java21-debian12:latest@sha256:7c9a9a362eadadb308d29b9c7fec2b39e5d5aa21d58837176a2cca50bdd06609 AS base

FROM maven:3.9.11-eclipse-temurin-21 AS maven-deps
COPY pom.xml /tmp/pom.xml
RUN cd /tmp && mvn -f pom.xml dependency:go-offline

FROM scratch
COPY --from=base --link / /
COPY --from=build --link --chmod=0755 --chown=root:root /app/reactor-grpc-protoc.jar .
COPY --from=maven-deps /root/.m2/repository /maven-repository
USER nobody
ENTRYPOINT [ "/usr/bin/java", "-jar", "/reactor-grpc-protoc.jar"]

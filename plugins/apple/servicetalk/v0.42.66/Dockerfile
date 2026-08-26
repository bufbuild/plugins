# syntax=docker/dockerfile:1.26
FROM debian:trixie-20260803@sha256:34cd9e9fd437c0a095ec39cb2e73422c9f30821b0d0848ed74fd0d43bae4d958 AS build

WORKDIR /app
RUN apt-get update \
 && apt-get install -y curl
RUN curl -fsSL -o servicetalk-grpc-protoc.jar https://repo1.maven.org/maven2/io/servicetalk/servicetalk-grpc-protoc/0.42.66/servicetalk-grpc-protoc-0.42.66-all.jar

FROM eclipse-temurin:25-jdk@sha256:e787e08ef76f4c16866108cd7f9fcd96a68eef3ac6cc76866897d4d02d5a2262 AS jre
RUN jlink --add-modules java.base,java.compiler,java.instrument,java.logging,java.management,jdk.unsupported \
  --strip-debug \
  --no-man-pages \
  --no-header-files \
  --generate-cds-archive \
  --output /jre \
 && rm /jre/lib/server/classes_nocoops.jsa

FROM gcr.io/distroless/cc-debian13:latest@sha256:e86cf4f565c8eee2cbb2be073bb107dafb14734b53d5872da20fdf47418a02f4 AS base

FROM maven:3.9.11-eclipse-temurin-21 AS maven-deps
COPY pom.xml /tmp/pom.xml
RUN cd /tmp && mvn -f pom.xml dependency:go-offline

FROM scratch
COPY --from=base --link / /
COPY --from=jre --link /jre /jre
COPY --from=build --link --chmod=0755 --chown=root:root /app/servicetalk-grpc-protoc.jar .
COPY --from=maven-deps /root/.m2/repository /maven-repository
USER nobody
ENTRYPOINT [ "/jre/bin/java", "-jar", "/servicetalk-grpc-protoc.jar"]

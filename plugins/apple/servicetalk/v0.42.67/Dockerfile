# syntax=docker/dockerfile:1.26
FROM debian:trixie-20260918@sha256:9cc080028c43b27d2074d63a5f9caf7166d731494965616c1a6d2827a004585c AS build

WORKDIR /app
RUN apt-get update \
 && apt-get install -y curl
RUN curl -fsSL -o servicetalk-grpc-protoc.jar https://repo1.maven.org/maven2/io/servicetalk/servicetalk-grpc-protoc/0.42.67/servicetalk-grpc-protoc-0.42.67-all.jar

FROM eclipse-temurin:25-jdk@sha256:119a3d18f160a3e7655a66034d0f43beee31cd7b3b9142d57a5de29772011de6 AS jre
RUN jlink --add-modules java.base,java.compiler,java.instrument,java.logging,java.management,jdk.unsupported \
  --strip-debug \
  --no-man-pages \
  --no-header-files \
  --generate-cds-archive \
  --output /jre \
 && rm /jre/lib/server/classes_nocoops.jsa

FROM gcr.io/distroless/cc-debian13:latest@sha256:4594d59540d1948417f6ca2829ddd9294493a7c68b7528f4dd459de7f203a750 AS base

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

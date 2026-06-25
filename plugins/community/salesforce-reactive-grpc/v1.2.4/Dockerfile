# syntax=docker/dockerfile:1.15
FROM debian:bookworm-20250610 AS build

WORKDIR /app
RUN apt-get update && apt-get install -y curl
RUN curl -fsSL -o reactor-grpc-protoc.jar https://repo1.maven.org/maven2/com/salesforce/servicelibs/reactor-grpc/1.2.4/reactor-grpc-1.2.4.jar

FROM eclipse-temurin:25-jdk@sha256:dfc0093e3dbf43dae57827111c6e374f5b44fac19a9451584b2b336b81474d64 AS jre
RUN jlink --add-modules java.base,java.compiler,java.instrument,java.logging,java.management,jdk.unsupported \
      --strip-debug --no-man-pages --no-header-files --output /jre

FROM gcr.io/distroless/cc-debian13:latest@sha256:a017e74bd2a12d98342dbecd33d121d2b160415ed777573dc1808969e989d94d AS base

FROM maven:3.9.11-eclipse-temurin-21 AS maven-deps
COPY pom.xml /tmp/pom.xml
RUN cd /tmp && mvn -f pom.xml dependency:go-offline

FROM scratch
COPY --from=base --link / /
COPY --from=jre --link /jre /jre
COPY --from=build --link --chmod=0755 --chown=root:root /app/reactor-grpc-protoc.jar .
COPY --from=maven-deps /root/.m2/repository /maven-repository
USER nobody
ENTRYPOINT [ "/jre/bin/java", "-jar", "/reactor-grpc-protoc.jar"]

# syntax=docker/dockerfile:1.10
FROM debian:bookworm-20241016 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl

#This script embeds the the .class files and is a self contained jvm protoc plugin. See https://scalapb.github.io/docs/scalapbc/#using-scalapb-as-a-proper-protoc-plugin for more details
RUN curl -fsSL -o protoc-gen-zio.jar https://repo1.maven.org/maven2/com/thesamet/scalapb/zio-grpc/protoc-gen-zio/0.6.3/protoc-gen-zio-0.6.3-unix.sh

FROM eclipse-temurin:25-jdk@sha256:dfc0093e3dbf43dae57827111c6e374f5b44fac19a9451584b2b336b81474d64 AS jre
RUN jlink --add-modules java.base,java.compiler,java.instrument,java.logging,java.management,jdk.unsupported \
      --strip-debug --no-man-pages --no-header-files --output /jre

FROM gcr.io/distroless/cc-debian13:latest@sha256:a017e74bd2a12d98342dbecd33d121d2b160415ed777573dc1808969e989d94d AS base

FROM scratch
COPY --from=base --link / /
COPY --from=jre --link /jre /jre
COPY --from=build --link /protoc-gen-zio.jar .
USER nobody
ENTRYPOINT ["/jre/bin/java", "-jar", "/protoc-gen-zio.jar"]

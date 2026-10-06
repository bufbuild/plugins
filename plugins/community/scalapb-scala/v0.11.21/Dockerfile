# syntax=docker/dockerfile:1.27
FROM debian:trixie-20260918@sha256:9cc080028c43b27d2074d63a5f9caf7166d731494965616c1a6d2827a004585c AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl

#This script embeds the the .class files and is a self contained jvm protoc plugin. See https://scalapb.github.io/docs/scalapbc/#using-scalapb-as-a-proper-protoc-plugin for more details
RUN curl -fsSL -o protoc-gen-scala.jar https://repo1.maven.org/maven2/com/thesamet/scalapb/protoc-gen-scala/0.11.21/protoc-gen-scala-0.11.21-unix.sh

FROM eclipse-temurin:25-jdk@sha256:8c0a84ea11c8f6ed52600fc19f1040121f2a162998e9f50a5faebbbad9172dcc AS jre
RUN jlink --add-modules java.base,java.compiler,java.instrument,java.logging,java.management,jdk.unsupported \
      --strip-debug --no-man-pages --no-header-files --output /jre

FROM gcr.io/distroless/cc-debian13:latest@sha256:159783207c2cd44c2aa5715961d13c8612368ac9bd450f887e3f08fc8ea461e3 AS base

FROM scratch
COPY --link --from=base / /
COPY --from=jre --link /jre /jre
COPY --link --from=build /protoc-gen-scala.jar .
USER nobody
ENTRYPOINT ["/jre/bin/java", "-jar", "/protoc-gen-scala.jar"]

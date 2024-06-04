# syntax=docker/dockerfile:1.7
FROM debian:bookworm-20240513 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl

#This script embeds the the .class files and is a self contained jvm protoc plugin. See https://scalapb.github.io/docs/scalapbc/#using-scalapb-as-a-proper-protoc-plugin for more details
RUN curl -fsSL -o protoc-gen-scala.jar https://repo1.maven.org/maven2/com/thesamet/scalapb/protoc-gen-scala/0.11.17/protoc-gen-scala-0.11.17-unix.sh

FROM gcr.io/distroless/java17-debian12:latest@sha256:1b2d30d74581a959f19704771ee5799160d3b8b7d076a94ca6c91c4065391eef AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build /protoc-gen-scala.jar .
USER nobody
ENTRYPOINT ["/usr/bin/java", "-jar", "/protoc-gen-scala.jar"]

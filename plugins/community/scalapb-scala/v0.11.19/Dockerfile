# syntax=docker/dockerfile:1.15
FROM debian:bookworm-20250610 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl

#This script embeds the the .class files and is a self contained jvm protoc plugin. See https://scalapb.github.io/docs/scalapbc/#using-scalapb-as-a-proper-protoc-plugin for more details
RUN curl -fsSL -o protoc-gen-scala.jar https://repo1.maven.org/maven2/com/thesamet/scalapb/protoc-gen-scala/0.11.19/protoc-gen-scala-0.11.19-unix.sh

FROM gcr.io/distroless/java21-debian12:latest@sha256:7c05bf8a64ff1a70a16083e9bdd35b463aa0d014c2fc782d31d13ea7a61de633 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build /protoc-gen-scala.jar .
USER nobody
ENTRYPOINT ["/usr/bin/java", "-jar", "/protoc-gen-scala.jar"]

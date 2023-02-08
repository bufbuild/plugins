# syntax=docker/dockerfile:1.4
FROM debian:bullseye-20230202 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl

#This script embeds the the .class files and is a self contained jvm protoc plugin. See https://scalapb.github.io/docs/scalapbc/#using-scalapb-as-a-proper-protoc-plugin for more details
RUN curl -fsSL -o protoc-gen-scala.jar https://repo1.maven.org/maven2/com/thesamet/scalapb/protoc-gen-scala/0.11.13/protoc-gen-scala-0.11.13-unix.sh

FROM gcr.io/distroless/java11-debian11
COPY --from=build --link /protoc-gen-scala.jar .
USER nobody
ENTRYPOINT [ "/usr/bin/java", "-jar", "/protoc-gen-scala.jar"]

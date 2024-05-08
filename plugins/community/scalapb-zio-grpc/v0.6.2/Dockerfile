# syntax=docker/dockerfile:1.4
FROM debian:bookworm-20240423 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl

#This script embeds the the .class files and is a self contained jvm protoc plugin. See https://scalapb.github.io/docs/scalapbc/#using-scalapb-as-a-proper-protoc-plugin for more details
RUN curl -fsSL -o protoc-gen-zio.jar https://repo1.maven.org/maven2/com/thesamet/scalapb/zio-grpc/protoc-gen-zio/0.6.2/protoc-gen-zio-0.6.2-unix.sh

FROM gcr.io/distroless/java17-debian12:latest@sha256:b43d7d66c2cd99b330c6ec6adf351acb447cce34c17b14dd84e04fb7c28b3ea9
COPY --from=build --link /protoc-gen-zio.jar .
USER nobody
ENTRYPOINT [ "/usr/bin/java", "-jar", "/protoc-gen-zio.jar"]

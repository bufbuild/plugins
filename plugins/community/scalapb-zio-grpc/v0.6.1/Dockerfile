# syntax=docker/dockerfile:1.4
FROM debian:bookworm-20231218 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl

#This script embeds the the .class files and is a self contained jvm protoc plugin. See https://scalapb.github.io/docs/scalapbc/#using-scalapb-as-a-proper-protoc-plugin for more details
RUN curl -fsSL -o protoc-gen-zio.jar https://repo1.maven.org/maven2/com/thesamet/scalapb/zio-grpc/protoc-gen-zio/0.6.1/protoc-gen-zio-0.6.1-unix.sh

FROM gcr.io/distroless/java17-debian12:latest@sha256:3da8c1021317fa0b0eca2d7bc9f54a1d09517fad07587abda0e1231bff207795
COPY --from=build --link /protoc-gen-zio.jar .
USER nobody
ENTRYPOINT [ "/usr/bin/java", "-jar", "/protoc-gen-zio.jar"]

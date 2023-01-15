# syntax=docker/dockerfile:1.4
FROM debian:bullseye-20230109 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl

#This script embeds the the .class files and is a self contained jvm protoc plugin. See https://scalapb.github.io/docs/scalapbc/#using-scalapb-as-a-proper-protoc-plugin for more details
RUN curl -fsSL -o protoc-gen-fs2-grpc.jar https://repo.maven.apache.org/maven2/org/typelevel/protoc-gen-fs2-grpc/2.5.6/protoc-gen-fs2-grpc-2.5.6-unix.sh

FROM gcr.io/distroless/java11-debian11
COPY --from=build --link /protoc-gen-fs2-grpc.jar .
USER nobody
ENTRYPOINT [ "/usr/bin/java", "-jar", "/protoc-gen-fs2-grpc.jar"]

# syntax=docker/dockerfile:1.4
FROM debian:bullseye-20230109 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl

#This script embeds the the .class files and is a self contained jvm protoc plugin. See https://scalapb.github.io/docs/scalapbc/#using-scalapb-as-a-proper-protoc-plugin for more details
RUN curl -fsSL -o protoc-gen-akka.jar https://repo1.maven.org/maven2/com/lightbend/akka/grpc/akka-grpc-codegen_2.13/2.2.1/akka-grpc-codegen_2.13-2.2.1-assembly.jar

FROM gcr.io/distroless/java11-debian11
COPY --from=build --link /protoc-gen-akka.jar .
USER nobody
ENTRYPOINT [ "/usr/bin/java", "-jar", "/protoc-gen-akka.jar"]

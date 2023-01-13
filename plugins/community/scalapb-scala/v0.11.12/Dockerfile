# syntax=docker/dockerfile:1.4
FROM debian:bullseye-20221114 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl

#This script embeds the the .class files and is a self contained jvm protoc plugin. See https://scalapb.github.io/docs/scalapbc/#using-scalapb-as-a-proper-protoc-plugin for more details
RUN curl -fsSL -o protoc-gen-scala.sh https://repo1.maven.org/maven2/com/thesamet/scalapb/protoc-gen-scala/0.11.12/protoc-gen-scala-0.11.12-unix.sh \
    && chmod +x protoc-gen-scala.sh

FROM gcr.io/distroless/java11-debian11
#We need shell access to run the script, we don't use the debug container as it contains more than what we need
COPY --from=build /bin/sh /bin/sh
COPY --from=build --link /protoc-gen-scala.sh .
USER nobody
ENTRYPOINT [ "ls" ]

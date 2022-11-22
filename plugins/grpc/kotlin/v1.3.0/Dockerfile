# syntax=docker/dockerfile:1.4
FROM debian:bullseye-20221114 AS build

ARG TARGETARCH

WORKDIR /build
RUN apt-get update \
 && apt-get install -y curl
RUN curl -fsSL -o protoc-gen-grpc-kotlin.jar https://repo1.maven.org/maven2/io/grpc/protoc-gen-grpc-kotlin/1.3.0/protoc-gen-grpc-kotlin-1.3.0-jdk8.jar

FROM gcr.io/distroless/java11-debian11
COPY --from=build --link --chmod=0644 --chown=root:root /build/protoc-gen-grpc-kotlin.jar .
USER nobody
ENTRYPOINT [ "/usr/bin/java", "-jar", "/protoc-gen-grpc-kotlin.jar" ]

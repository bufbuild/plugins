# syntax=docker/dockerfile:1.19
FROM debian:bookworm-20251103 AS build

WORKDIR /app
RUN apt-get update \
 && apt-get install -y curl
RUN curl -fsSL -o servicetalk-grpc-protoc.jar https://repo1.maven.org/maven2/io/servicetalk/servicetalk-grpc-protoc/0.42.61/servicetalk-grpc-protoc-0.42.61-all.jar

FROM gcr.io/distroless/java21-debian12:latest@sha256:b41ca849c90e111ed5a6d2431b474225535f266ac1b3902cd508718f160cea60 AS base

FROM scratch
COPY --from=base --link / /
COPY --from=build --link --chmod=0755 --chown=root:root /app/servicetalk-grpc-protoc.jar .
USER nobody
ENTRYPOINT [ "/usr/bin/java", "-jar", "/servicetalk-grpc-protoc.jar"]

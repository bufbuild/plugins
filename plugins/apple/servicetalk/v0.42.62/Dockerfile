# syntax=docker/dockerfile:1.19
FROM debian:bookworm-20251229 AS build

WORKDIR /app
RUN apt-get update \
 && apt-get install -y curl
RUN curl -fsSL -o servicetalk-grpc-protoc.jar https://repo1.maven.org/maven2/io/servicetalk/servicetalk-grpc-protoc/0.42.62/servicetalk-grpc-protoc-0.42.62-all.jar

FROM gcr.io/distroless/java21-debian12:latest@sha256:ed87b011df38601c55503cb24a0d136fed216aeb3bcd57925719488d93d236f4 AS base

FROM scratch
COPY --from=base --link / /
COPY --from=build --link --chmod=0755 --chown=root:root /app/servicetalk-grpc-protoc.jar .
USER nobody
ENTRYPOINT [ "/usr/bin/java", "-jar", "/servicetalk-grpc-protoc.jar"]

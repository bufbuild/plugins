# syntax=docker/dockerfile:1.18
FROM debian:bookworm-20250908 AS build

WORKDIR /app
RUN apt-get update \
 && apt-get install -y curl
RUN curl -fsSL -o servicetalk-grpc-protoc.jar https://repo1.maven.org/maven2/io/servicetalk/servicetalk-grpc-protoc/0.42.60/servicetalk-grpc-protoc-0.42.60-all.jar

FROM gcr.io/distroless/java21-debian12:latest@sha256:418b2e2a9e452aa9299511427f2ae404dfc910ecfa78feb53b1c60c22c3b640c AS base

FROM scratch
COPY --from=base --link / /
COPY --from=build --link --chmod=0755 --chown=root:root /app/servicetalk-grpc-protoc.jar .
USER nobody
ENTRYPOINT [ "/usr/bin/java", "-jar", "/servicetalk-grpc-protoc.jar"]

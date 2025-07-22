# syntax=docker/dockerfile:1.15
FROM debian:bookworm-20250721 AS build

WORKDIR /app
RUN apt-get update \
 && apt-get install -y curl
RUN curl -fsSL -o servicetalk-grpc-protoc.jar https://repo1.maven.org/maven2/io/servicetalk/servicetalk-grpc-protoc/0.42.58/servicetalk-grpc-protoc-0.42.58-all.jar

FROM gcr.io/distroless/java21-debian12:latest@sha256:73c719485ac6fb38dd168053837b95812d33882d7136d03e56291bb0e1c13bf8 AS base

CMD echo this is a dummy file used to automate dependency upgrades for plugins

FROM scratch
COPY --from=base --link / /
COPY --from=build --link --chmod=0755 --chown=root:root /app/servicetalk-grpc-protoc.jar .
USER nobody
ENTRYPOINT [ "/usr/bin/java", "-jar", "/servicetalk-grpc-protoc.jar"]

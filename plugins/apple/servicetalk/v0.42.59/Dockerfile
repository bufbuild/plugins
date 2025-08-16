# syntax=docker/dockerfile:1.17
FROM debian:bookworm-20250811 AS build

WORKDIR /app
RUN apt-get update \
 && apt-get install -y curl
RUN curl -fsSL -o servicetalk-grpc-protoc.jar https://repo1.maven.org/maven2/io/servicetalk/servicetalk-grpc-protoc/0.42.59/servicetalk-grpc-protoc-0.42.59-all.jar

FROM gcr.io/distroless/java21-debian12:latest@sha256:914d2e4d0aef6afe6167a11de8d87a4bfcd9325f36d1b45c03c04e6f16ba94d8 AS base

CMD echo this is a dummy file used to automate dependency upgrades for plugins

FROM scratch
COPY --from=base --link / /
COPY --from=build --link --chmod=0755 --chown=root:root /app/servicetalk-grpc-protoc.jar .
USER nobody
ENTRYPOINT [ "/usr/bin/java", "-jar", "/servicetalk-grpc-protoc.jar"]

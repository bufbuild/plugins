# syntax=docker/dockerfile:1.15
FROM debian:bookworm-20250520 AS build

WORKDIR /app
RUN apt-get update && apt-get install -y curl
RUN curl -fsSL -o reactor-grpc-protoc.jar https://repo1.maven.org/maven2/com/salesforce/servicelibs/reactor-grpc/1.2.4/reactor-grpc-1.2.4.jar

FROM gcr.io/distroless/java21-debian12:latest@sha256:7c9a9a362eadadb308d29b9c7fec2b39e5d5aa21d58837176a2cca50bdd06609 AS base

CMD echo this is a dummy file used to automate dependency upgrades for plugins

FROM scratch
COPY --from=base --link / /
COPY --from=build --link --chmod=0755 --chown=root:root /app/reactor-grpc-protoc.jar .
USER nobody
ENTRYPOINT [ "/usr/bin/java", "-jar", "/reactor-grpc-protoc.jar"]

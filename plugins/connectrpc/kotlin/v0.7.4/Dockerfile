# syntax=docker/dockerfile:1.15
FROM debian:bookworm-20250630 AS build

RUN apt-get update \
    && apt-get install -y curl
WORKDIR /app
RUN curl -fsSL -o /app/protoc-gen-connect-kotlin.jar https://repo1.maven.org/maven2/com/connectrpc/protoc-gen-connect-kotlin/0.7.4/protoc-gen-connect-kotlin-0.7.4.jar

FROM gcr.io/distroless/java21-debian12:latest@sha256:7c05bf8a64ff1a70a16083e9bdd35b463aa0d014c2fc782d31d13ea7a61de633 as base

FROM scratch
COPY --from=base --link / /
COPY --from=build --link --chmod=0755 --chown=root:root /app/protoc-gen-connect-kotlin.jar .
USER nobody
ENTRYPOINT [ "/usr/bin/java", "-jar", "/protoc-gen-connect-kotlin.jar"]

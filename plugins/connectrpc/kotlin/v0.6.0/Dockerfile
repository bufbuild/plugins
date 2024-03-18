# syntax=docker/dockerfile:1.4
FROM debian:bookworm-20240311 AS build
RUN apt-get update \
 && apt-get install -y curl
WORKDIR /app
RUN curl -fsSL -o /app/protoc-gen-connect-kotlin.jar https://repo1.maven.org/maven2/com/connectrpc/protoc-gen-connect-kotlin/0.6.0/protoc-gen-connect-kotlin-0.6.0.jar

FROM gcr.io/distroless/java17-debian12:latest@sha256:9008a52842724ed0d6fa5c87050a8477d11656449f5f5acef961cade3e3e46a3
WORKDIR /app
COPY --from=build /app/protoc-gen-connect-kotlin.jar /app
CMD ["/app/protoc-gen-connect-kotlin.jar"]

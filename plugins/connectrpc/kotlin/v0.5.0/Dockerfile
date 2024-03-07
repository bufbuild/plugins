# syntax=docker/dockerfile:1.4
FROM debian:bookworm-20240211 AS build
RUN apt-get update \
 && apt-get install -y curl
WORKDIR /app
RUN curl -fsSL -o /app/protoc-gen-connect-kotlin.jar https://repo1.maven.org/maven2/com/connectrpc/protoc-gen-connect-kotlin/0.5.0/protoc-gen-connect-kotlin-0.5.0.jar

FROM gcr.io/distroless/java17-debian12:latest@sha256:2ed10f986506097fb476fe994f97fbb1d73275a577febc3de65ecf0f73f14651
WORKDIR /app
COPY --from=build /app/protoc-gen-connect-kotlin.jar /app
CMD ["/app/protoc-gen-connect-kotlin.jar"]

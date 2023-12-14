# syntax=docker/dockerfile:1.4
FROM debian:bookworm-20231120 AS build
RUN apt-get update \
 && apt-get install -y curl
WORKDIR /app
RUN curl -fsSL -o /app/protoc-gen-connect-kotlin.jar https://repo1.maven.org/maven2/com/connectrpc/protoc-gen-connect-kotlin/0.4.0/protoc-gen-connect-kotlin-0.4.0.jar

FROM gcr.io/distroless/java17-debian12:latest@sha256:76ac1768c577881e3cc04a4da5bdfa650c1ae5129ecdc9b5367d61881eb05b79
WORKDIR /app
COPY --from=build /app/protoc-gen-connect-kotlin.jar /app
CMD ["/app/protoc-gen-connect-kotlin.jar"]

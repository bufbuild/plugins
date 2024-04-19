# syntax=docker/dockerfile:1.7
FROM debian:bookworm-20240408 AS build
RUN apt-get update \
 && apt-get install -y curl
WORKDIR /app
RUN curl -fsSL -o /app/protoc-gen-connect-kotlin.jar https://repo1.maven.org/maven2/com/connectrpc/protoc-gen-connect-kotlin/0.6.1/protoc-gen-connect-kotlin-0.6.1.jar

FROM gcr.io/distroless/java17-debian12:latest@sha256:9dc2b13d8adb97617c308a4150f7af193ef4acbdd74317767ceb13ea13a16055
WORKDIR /app
COPY --from=build /app/protoc-gen-connect-kotlin.jar /app
CMD ["/app/protoc-gen-connect-kotlin.jar"]

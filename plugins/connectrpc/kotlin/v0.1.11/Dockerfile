# syntax=docker/dockerfile:1.4
FROM debian:bullseye-20230904 AS build
RUN apt-get update \
 && apt-get install -y curl
WORKDIR /app
RUN curl -fsSL -o /app/protoc-gen-connect-kotlin.jar https://repo1.maven.org/maven2/com/connectrpc/protoc-gen-connect-kotlin/0.1.11/protoc-gen-connect-kotlin-0.1.11.jar

FROM gcr.io/distroless/java17-debian11
WORKDIR /app
COPY --from=build /app/protoc-gen-connect-kotlin.jar /app
CMD ["/app/protoc-gen-connect-kotlin.jar"]

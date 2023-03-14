# syntax=docker/dockerfile:1.4
FROM debian:bullseye-20230227 AS build
RUN apt-get update \
 && apt-get install -y curl
WORKDIR /app
RUN curl -fsSL -o /app/protoc-gen-connect-kotlin.jar https://github.com/bufbuild/connect-kotlin/releases/download/v0.1.4/protoc-gen-connect-kotlin.jar

FROM gcr.io/distroless/java17-debian11
WORKDIR /app
COPY --from=build /app/protoc-gen-connect-kotlin.jar /app
CMD ["/app/protoc-gen-connect-kotlin.jar"]

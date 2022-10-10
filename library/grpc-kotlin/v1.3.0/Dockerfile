# syntax=docker/dockerfile:1.4
FROM debian:bullseye-20220822-slim AS build

ARG TARGETARCH

WORKDIR /build
RUN apt-get update \
 && apt-get install -y curl
RUN curl -fsSL -o protoc-gen-grpc-kotlin.jar https://repo1.maven.org/maven2/io/grpc/protoc-gen-grpc-kotlin/1.3.0/protoc-gen-grpc-kotlin-1.3.0-jdk8.jar

FROM eclipse-temurin:18.0.2_9-jre-jammy
COPY --from=build --link --chmod=0644 --chown=root:root /build/protoc-gen-grpc-kotlin.jar .
COPY --chmod=0755 --link --chown=root:root <<'EOF' /entrypoint.sh
#!/bin/bash
exec java -jar /protoc-gen-grpc-kotlin.jar "$@"
EOF
USER nobody
ENTRYPOINT [ "/entrypoint.sh" ]

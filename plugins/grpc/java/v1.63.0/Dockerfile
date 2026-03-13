# syntax=docker/dockerfile:1.7
FROM debian:bookworm-20240311 AS build

ARG TARGETARCH

WORKDIR /build
RUN apt-get update \
 && apt-get install -y curl
RUN arch=${TARGETARCH}; \
    if [ "${arch}" = "arm64" ]; then\
        arch="aarch_64"; \
    elif [ "${arch}" = "amd64" ]; then\
        arch="x86_64"; \
    fi; \
    echo "${arch}"; \
    curl -fsSL -o protoc-gen-grpc-java https://repo1.maven.org/maven2/io/grpc/protoc-gen-grpc-java/1.63.0/protoc-gen-grpc-java-1.63.0-linux-${arch}.exe

FROM maven:3.9.11-eclipse-temurin-21 AS maven-deps
COPY pom.xml /tmp/pom.xml
RUN cd /tmp && mvn -f pom.xml dependency:copy-dependencies -Dmdep.useRepositoryLayout=true -Dmdep.copyPom=true -DoutputDirectory=/maven-repository && cd /root/.m2/repository && find . -name '*.pom' -exec cp --parents -t /maven-repository {} +

FROM gcr.io/distroless/base-debian12:latest@sha256:611d30d7f6d9992c37b1e1a212eefdf1f7c671deb56db3707e24eb01da8c4c2a
COPY --from=build --link --chmod=0755 --chown=root:root /build/protoc-gen-grpc-java .
COPY --from=maven-deps /maven-repository /maven-repository
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-java" ]

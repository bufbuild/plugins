# syntax=docker/dockerfile:1.19
FROM debian:bookworm-20260202 AS build

WORKDIR /app
RUN apt-get update \
 && apt-get install -y curl
RUN curl -fsSL -o servicetalk-grpc-protoc.jar https://repo1.maven.org/maven2/io/servicetalk/servicetalk-grpc-protoc/0.42.63/servicetalk-grpc-protoc-0.42.63-all.jar

FROM gcr.io/distroless/java21-debian12:latest@sha256:f34fd3e4e2d7a246d764d0614f5e6ffb3a735930723fac4cfc25a72798950262 AS base

FROM maven:3.9.11-eclipse-temurin-21 AS maven-deps
COPY pom.xml /tmp/pom.xml
RUN cd /tmp && mvn -f pom.xml dependency:copy-dependencies -Dmdep.useRepositoryLayout=true -Dmdep.copyPom=true -DoutputDirectory=/maven-repository && cd /root/.m2/repository && find . -name '*.pom' -exec sh -c 'mkdir -p "/maven-repository/$(dirname "$1")" && cp -n "$1" "/maven-repository/$1"' _ {} \;

FROM scratch
COPY --from=base --link / /
COPY --from=build --link --chmod=0755 --chown=root:root /app/servicetalk-grpc-protoc.jar .
COPY --from=maven-deps /maven-repository /maven-repository
USER nobody
ENTRYPOINT [ "/usr/bin/java", "-jar", "/servicetalk-grpc-protoc.jar"]

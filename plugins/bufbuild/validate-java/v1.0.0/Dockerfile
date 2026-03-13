# syntax=docker/dockerfile:1.4
FROM golang:1.20.3-bullseye AS build
RUN CGO_ENABLED=0 go install -ldflags "-s -w" -trimpath github.com/envoyproxy/protoc-gen-validate/cmd/protoc-gen-validate-java@v1.0.0

FROM maven:3.9.11-eclipse-temurin-21 AS maven-deps
COPY pom.xml /tmp/pom.xml
RUN cd /tmp && mvn -f pom.xml dependency:go-offline

FROM scratch
COPY --from=build --link /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-validate-java .
COPY --from=maven-deps /root/.m2/repository /maven-repository
USER nobody
ENTRYPOINT [ "/protoc-gen-validate-java" ]

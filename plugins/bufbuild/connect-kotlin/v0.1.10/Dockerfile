# syntax=docker/dockerfile:1.4
FROM debian:bullseye-20230904 AS build
RUN apt-get update \
 && apt-get install -y curl
WORKDIR /app
RUN curl -fsSL -o /app/protoc-gen-connect-kotlin.jar https://repo1.maven.org/maven2/build/buf/protoc-gen-connect-kotlin/0.1.10/protoc-gen-connect-kotlin-0.1.10.jar

FROM maven:3.9.11-eclipse-temurin-21 AS maven-deps
COPY pom.xml /tmp/pom.xml
RUN cd /tmp && mvn -f pom.xml dependency:copy-dependencies -Dmdep.useRepositoryLayout=true -Dmdep.copyPom=true -DoutputDirectory=/maven-repository && cd /root/.m2/repository && find . -name '*.pom' -exec sh -c 'mkdir -p "/maven-repository/$(dirname "$1")" && cp -n "$1" "/maven-repository/$1"' _ {} \;

FROM gcr.io/distroless/java17-debian11
WORKDIR /app
COPY --from=build /app/protoc-gen-connect-kotlin.jar /app
COPY --from=maven-deps /maven-repository /maven-repository
CMD ["/app/protoc-gen-connect-kotlin.jar"]

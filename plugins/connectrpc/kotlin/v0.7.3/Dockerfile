# syntax=docker/dockerfile:1.15
FROM debian:bookworm-20250428 AS build

RUN apt-get update \
    && apt-get install -y curl
WORKDIR /app
RUN curl -fsSL -o /app/protoc-gen-connect-kotlin.jar https://repo1.maven.org/maven2/com/connectrpc/protoc-gen-connect-kotlin/0.7.3/protoc-gen-connect-kotlin-0.7.3.jar

FROM maven:3.9.11-eclipse-temurin-25 AS maven-deps
COPY <<EOF /tmp/pom.xml
<project>
  <modelVersion>4.0.0</modelVersion>
  <groupId>temp</groupId>
  <artifactId>temp</artifactId>
  <version>1.0</version>
  <dependencies>
    <dependency>
      <groupId>com.connectrpc</groupId>
      <artifactId>connect-kotlin</artifactId>
      <version>0.7.3</version>
    </dependency>
    <dependency>
      <groupId>com.connectrpc</groupId>
      <artifactId>connect-kotlin-google-java-ext</artifactId>
      <version>0.7.3</version>
    </dependency>
    <dependency>
      <groupId>com.connectrpc</groupId>
      <artifactId>connect-kotlin-okhttp</artifactId>
      <version>0.7.3</version>
    </dependency>
    <dependency>
      <groupId>com.connectrpc</groupId>
      <artifactId>connect-kotlin-google-javalite-ext</artifactId>
      <version>0.7.3</version>
    </dependency>
  </dependencies>
</project>
EOF
RUN cd /tmp && mvn -f pom.xml dependency:go-offline

FROM gcr.io/distroless/java21-debian12:latest@sha256:c298bfc8c8b1aa3d7b03480dcf52001a90d66d966f6a8d8997ae837d3982be3f as base

FROM scratch
COPY --from=base --link / /
COPY --from=build --link --chmod=0755 --chown=root:root /app/protoc-gen-connect-kotlin.jar .
COPY --from=maven-deps /root/.m2/repository /maven-repository
USER nobody
ENTRYPOINT [ "/usr/bin/java", "-jar", "/protoc-gen-connect-kotlin.jar"]

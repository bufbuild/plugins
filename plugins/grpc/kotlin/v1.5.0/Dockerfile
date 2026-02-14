# syntax=docker/dockerfile:1.18
FROM debian:bookworm-20250908 AS build

ARG TARGETARCH

WORKDIR /build
RUN apt-get update \
 && apt-get install -y curl
RUN curl -fsSL -o protoc-gen-grpc-kotlin.jar https://repo1.maven.org/maven2/io/grpc/protoc-gen-grpc-kotlin/1.5.0/protoc-gen-grpc-kotlin-1.5.0-jdk8.jar

FROM gcr.io/distroless/java21-debian12:latest@sha256:418b2e2a9e452aa9299511427f2ae404dfc910ecfa78feb53b1c60c22c3b640c AS base

FROM maven:3.9.11-eclipse-temurin-25 AS maven-deps
COPY <<EOF /tmp/pom.xml
<project>
  <modelVersion>4.0.0</modelVersion>
  <groupId>temp</groupId>
  <artifactId>temp</artifactId>
  <version>1.0</version>
  <dependencies>
    <dependency>
      <groupId>io.grpc</groupId>
      <artifactId>grpc-kotlin-stub</artifactId>
      <version>1.5.0</version>
    </dependency>
    <dependency>
      <groupId>org.jetbrains.kotlinx</groupId>
      <artifactId>kotlinx-coroutines-core-jvm</artifactId>
      <version>1.10.2</version>
    </dependency>
    <dependency>
      <groupId>org.jetbrains.kotlin</groupId>
      <artifactId>kotlin-compiler-embeddable</artifactId>
      <version>2.2.20</version>
    </dependency>
    <dependency>
      <groupId>org.jetbrains.kotlin</groupId>
      <artifactId>kotlin-scripting-compiler</artifactId>
      <version>2.2.20</version>
    </dependency>
  </dependencies>
  <build>
    <plugins>
      <plugin>
        <groupId>org.jetbrains.kotlin</groupId>
        <artifactId>kotlin-maven-plugin</artifactId>
        <version>2.2.20</version>
        <configuration>
        </configuration>
      </plugin>
      <plugin>
        <groupId>org.apache.maven.plugins</groupId>
        <artifactId>maven-compiler-plugin</artifactId>
        <version>3.12.1</version>
      </plugin>
      <plugin>
        <groupId>org.apache.maven.plugins</groupId>
        <artifactId>maven-source-plugin</artifactId>
        <version>3.3.0</version>
      </plugin>
    </plugins>
  </build>
</project>
EOF
RUN cd /tmp && mvn -f pom.xml dependency:go-offline

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0644 --chown=root:root /build/protoc-gen-grpc-kotlin.jar .
COPY --from=maven-deps /root/.m2/repository /maven-repository
USER nobody
ENTRYPOINT [ "/usr/bin/java", "-jar", "/protoc-gen-grpc-kotlin.jar" ]

# syntax=docker/dockerfile:1.19
FROM debian:bookworm-20251103 AS build

WORKDIR /app
RUN apt-get update \
 && apt-get install -y curl
RUN curl -fsSL -o servicetalk-grpc-protoc.jar https://repo1.maven.org/maven2/io/servicetalk/servicetalk-grpc-protoc/0.42.61/servicetalk-grpc-protoc-0.42.61-all.jar

FROM gcr.io/distroless/java21-debian12:latest@sha256:b41ca849c90e111ed5a6d2431b474225535f266ac1b3902cd508718f160cea60 AS base


FROM maven:3.9.11-eclipse-temurin-25 AS maven-deps
COPY <<EOF /tmp/pom.xml
<project>
  <modelVersion>4.0.0</modelVersion>
  <groupId>temp</groupId>
  <artifactId>temp</artifactId>
  <version>1.0</version>
  <dependencies>
    <dependency>
      <groupId>io.servicetalk</groupId>
      <artifactId>servicetalk-data-protobuf</artifactId>
      <version>0.42.61</version>
    </dependency>
    <dependency>
      <groupId>io.servicetalk</groupId>
      <artifactId>servicetalk-grpc-api</artifactId>
      <version>0.42.61</version>
    </dependency>
    <dependency>
      <groupId>io.servicetalk</groupId>
      <artifactId>servicetalk-grpc-protobuf</artifactId>
      <version>0.42.61</version>
    </dependency>
    <dependency>
      <groupId>com.google.protobuf</groupId>
      <artifactId>protobuf-java</artifactId>
      <version>4.33.1</version>
    </dependency>
  </dependencies>
  <build>
    <plugins>
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
COPY --from=base --link / /
COPY --from=build --link --chmod=0755 --chown=root:root /app/servicetalk-grpc-protoc.jar .
COPY --from=maven-deps /root/.m2/repository /maven-repository
USER nobody
ENTRYPOINT [ "/usr/bin/java", "-jar", "/servicetalk-grpc-protoc.jar"]

# syntax=docker/dockerfile:1.15
FROM debian:bookworm-20250721 AS build

WORKDIR /app
RUN apt-get update \
 && apt-get install -y curl
RUN curl -fsSL -o servicetalk-grpc-protoc.jar https://repo1.maven.org/maven2/io/servicetalk/servicetalk-grpc-protoc/0.42.58/servicetalk-grpc-protoc-0.42.58-all.jar

FROM gcr.io/distroless/java21-debian12:latest@sha256:73c719485ac6fb38dd168053837b95812d33882d7136d03e56291bb0e1c13bf8 AS base

CMD echo this is a dummy file used to automate dependency upgrades for plugins


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
      <version>0.42.58</version>
    </dependency>
    <dependency>
      <groupId>io.servicetalk</groupId>
      <artifactId>servicetalk-grpc-api</artifactId>
      <version>0.42.58</version>
    </dependency>
    <dependency>
      <groupId>io.servicetalk</groupId>
      <artifactId>servicetalk-grpc-protobuf</artifactId>
      <version>0.42.58</version>
    </dependency>
    <dependency>
      <groupId>com.google.protobuf</groupId>
      <artifactId>protobuf-java</artifactId>
      <version>4.31.1</version>
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

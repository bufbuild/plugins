# syntax=docker/dockerfile:1.8
FROM debian:bookworm-20240701 AS build

RUN apt-get update \
    && apt-get install -y curl
WORKDIR /app
RUN curl -fsSL -o /app/protoc-gen-connect-kotlin.jar https://repo1.maven.org/maven2/com/connectrpc/protoc-gen-connect-kotlin/0.7.0/protoc-gen-connect-kotlin-0.7.0.jar

FROM gcr.io/distroless/java17-debian12:latest@sha256:c8c2c236c63d7652b707a05b02d0259db91d5feb357e9dc53942f73c2e001088 as base


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
      <version>0.7.0</version>
    </dependency>
    <dependency>
      <groupId>com.connectrpc</groupId>
      <artifactId>connect-kotlin-google-java-ext</artifactId>
      <version>0.7.0</version>
    </dependency>
    <dependency>
      <groupId>com.connectrpc</groupId>
      <artifactId>connect-kotlin-okhttp</artifactId>
      <version>0.7.0</version>
    </dependency>
    <dependency>
      <groupId>com.connectrpc</groupId>
      <artifactId>connect-kotlin</artifactId>
      <version>0.7.0</version>
    </dependency>
    <dependency>
      <groupId>com.connectrpc</groupId>
      <artifactId>connect-kotlin-google-javalite-ext</artifactId>
      <version>0.7.0</version>
    </dependency>
    <dependency>
      <groupId>com.connectrpc</groupId>
      <artifactId>connect-kotlin-okhttp</artifactId>
      <version>0.7.0</version>
    </dependency>
    <dependency>
      <groupId>org.jetbrains.kotlin</groupId>
      <artifactId>kotlin-compiler-embeddable</artifactId>
      <version>1.8.22</version>
    </dependency>
    <dependency>
      <groupId>org.jetbrains.kotlin</groupId>
      <artifactId>kotlin-scripting-compiler</artifactId>
      <version>1.8.22</version>
    </dependency>
  </dependencies>
  <build>
    <plugins>
      <plugin>
        <groupId>org.jetbrains.kotlin</groupId>
        <artifactId>kotlin-maven-plugin</artifactId>
        <version>1.8.22</version>
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
COPY --from=base --link / /
COPY --from=build --link --chmod=0755 --chown=root:root /app/protoc-gen-connect-kotlin.jar .
COPY --from=maven-deps /root/.m2/repository /maven-repository
USER nobody
ENTRYPOINT [ "/usr/bin/java", "-jar", "/protoc-gen-connect-kotlin.jar"]

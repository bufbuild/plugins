# syntax=docker/dockerfile:1.4
FROM debian:bullseye-20230814 AS build
RUN apt-get update \
 && apt-get install -y curl
WORKDIR /app
RUN curl -fsSL -o /app/protoc-gen-connect-kotlin.jar https://repo1.maven.org/maven2/build/buf/protoc-gen-connect-kotlin/0.1.9/protoc-gen-connect-kotlin-0.1.9.jar


FROM maven:3.9.11-eclipse-temurin-25 AS maven-deps
COPY <<EOF /tmp/pom.xml
<project>
  <modelVersion>4.0.0</modelVersion>
  <groupId>temp</groupId>
  <artifactId>temp</artifactId>
  <version>1.0</version>
  <dependencies>
    <dependency>
      <groupId>build.buf</groupId>
      <artifactId>connect-kotlin</artifactId>
      <version>0.1.9</version>
    </dependency>
    <dependency>
      <groupId>build.buf</groupId>
      <artifactId>connect-kotlin-google-java-ext</artifactId>
      <version>0.1.9</version>
    </dependency>
    <dependency>
      <groupId>build.buf</groupId>
      <artifactId>connect-kotlin-okhttp</artifactId>
      <version>0.1.9</version>
    </dependency>
    <dependency>
      <groupId>build.buf</groupId>
      <artifactId>connect-kotlin</artifactId>
      <version>0.1.9</version>
    </dependency>
    <dependency>
      <groupId>build.buf</groupId>
      <artifactId>connect-kotlin-google-javalite-ext</artifactId>
      <version>0.1.9</version>
    </dependency>
    <dependency>
      <groupId>build.buf</groupId>
      <artifactId>connect-kotlin-okhttp</artifactId>
      <version>0.1.9</version>
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

FROM gcr.io/distroless/java17-debian11
WORKDIR /app
COPY --from=build /app/protoc-gen-connect-kotlin.jar /app
COPY --from=maven-deps /root/.m2/repository /maven-repository
CMD ["/app/protoc-gen-connect-kotlin.jar"]

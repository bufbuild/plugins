# syntax=docker/dockerfile:1.19
FROM golang:1.26.0-bookworm AS build
RUN CGO_ENABLED=0 go install -ldflags "-s -w" -trimpath github.com/envoyproxy/protoc-gen-validate/cmd/protoc-gen-validate-java@v1.3.3


FROM maven:3.9.11-eclipse-temurin-25 AS maven-deps
COPY <<EOF /tmp/pom.xml
<project>
  <modelVersion>4.0.0</modelVersion>
  <groupId>temp</groupId>
  <artifactId>temp</artifactId>
  <version>1.0</version>
  <dependencies>
    <dependency>
      <groupId>build.buf.protoc-gen-validate</groupId>
      <artifactId>pgv-java-stub</artifactId>
      <version>1.3.3</version>
    </dependency>
    <dependency>
      <groupId>com.google.protobuf</groupId>
      <artifactId>protobuf-java</artifactId>
      <version>4.33.5</version>
    </dependency>
    <!-- lite -->
    <dependency>
      <groupId>com.google.protobuf</groupId>
      <artifactId>protobuf-javalite</artifactId>
      <version>4.33.5</version>
    </dependency>
    <dependency>
      <groupId>build.buf</groupId>
      <artifactId>protobuf-javalite</artifactId>
      <version>4.33.5</version>
    </dependency>
  </dependencies>
</project>
EOF
RUN cd /tmp && mvn -f pom.xml dependency:go-offline

FROM scratch
COPY --from=build --link /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-validate-java .
COPY --from=maven-deps /root/.m2/repository /maven-repository
USER nobody
ENTRYPOINT [ "/protoc-gen-validate-java" ]

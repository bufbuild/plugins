# syntax=docker/dockerfile:1.19
FROM debian:bookworm-20260112 AS build

ARG TARGETARCH
ARG BAZEL_OPTS="--host_jvm_args=-Djava.net.preferIPv4Stack=true"

RUN apt-get update \
 && apt-get install -y curl git cmake build-essential autoconf clang libc++-dev libtool pkg-config unzip zip
RUN curl -fsSL -o /usr/local/bin/bazelisk https://github.com/bazelbuild/bazelisk/releases/download/v1.25.0/bazelisk-linux-${TARGETARCH} \
 && chmod +x /usr/local/bin/bazelisk \
 && mkdir /build \
 && chown nobody:nogroup /build \
 && usermod --home /build nobody

USER nobody
WORKDIR /build
RUN curl -fsSL -o protoc.tar.gz https://github.com/protocolbuffers/protobuf/releases/download/v33.5/protobuf-33.5.tar.gz \
 && tar --strip-components=1 -zxf protoc.tar.gz \
 && rm protoc.tar.gz
RUN bazelisk ${BAZEL_OPTS} build '//:protoc_lib'
COPY --link BUILD java.cc plugins/
RUN bazelisk ${BAZEL_OPTS} build '//plugins:protoc-gen-java.stripped'

FROM gcr.io/distroless/cc-debian12:latest@sha256:72344f7f909a8bf003c67f55687e6d51a441b49661af8f660aa7b285f00e57df AS base

FROM maven:3.9.11-eclipse-temurin-25 AS maven-deps
COPY <<EOF /tmp/pom.xml
<project>
  <modelVersion>4.0.0</modelVersion>
  <groupId>temp</groupId>
  <artifactId>temp</artifactId>
  <version>1.0</version>
  <dependencies>
    <dependency>
      <groupId>com.google.protobuf</groupId>
      <artifactId>protobuf-java</artifactId>
      <version>4.33.5</version>
    </dependency>
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
COPY --from=build --link --chmod=0755 /build/bazel-bin/plugins/protoc-gen-java .
COPY --from=maven-deps /root/.m2/repository /maven-repository
USER nobody
ENTRYPOINT ["/protoc-gen-java"]

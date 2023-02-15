# syntax=docker/dockerfile:1.4
FROM gradle:7.4.0-jdk11 AS build
COPY --chown=gradle:gradle . /home/gradle/src
WORKDIR /home/gradle/src
USER root
RUN chown -R gradle /home/gradle/src

WORKDIR /tmp
RUN git clone https://github.com/bufbuild/connect-kotlin.git

WORKDIR /tmp/connect-kotlin
RUN gradle protoc-gen-connect-kotlin:jar --no-daemon

COPY protoc-gen-connect-kotlin/build/libs/protoc-gen-connect-kotlin.jar /connect-kotlin.jar
COPY protoc-gen-connect-kotlin/protoc-gen-connect-kotlin-remote /protoc-gen-connect-kotlin

ENTRYPOINT ["/protoc-gen-connect-kotlin"]

# syntax=docker/dockerfile:1.4
FROM debian:bullseye-20230904 AS build
RUN apt-get update \
 && apt-get install -y curl
WORKDIR /app
RUN curl -fsSL -o /app/protoc-gen-connect-kotlin.jar https://repo1.maven.org/maven2/build/buf/protoc-gen-connect-kotlin/0.1.10/protoc-gen-connect-kotlin-0.1.10.jar

FROM maven:3.9.11-eclipse-temurin-21 AS maven-deps
COPY pom.xml /tmp/pom.xml
RUN cd /tmp && mvn -f pom.xml dependency:go-offline

FROM eclipse-temurin:25-jdk@sha256:dfc0093e3dbf43dae57827111c6e374f5b44fac19a9451584b2b336b81474d64 AS jre
RUN jlink --add-modules java.base,java.compiler,java.instrument,java.logging,java.management,jdk.unsupported \
      --strip-debug --no-man-pages --no-header-files --output /jre

FROM gcr.io/distroless/cc-debian13:latest@sha256:a017e74bd2a12d98342dbecd33d121d2b160415ed777573dc1808969e989d94d
WORKDIR /app
COPY --from=jre --link /jre /jre
COPY --from=build /app/protoc-gen-connect-kotlin.jar /app
COPY --from=maven-deps /root/.m2/repository /maven-repository
ENTRYPOINT ["/jre/bin/java", "-jar", "/app/protoc-gen-connect-kotlin.jar"]

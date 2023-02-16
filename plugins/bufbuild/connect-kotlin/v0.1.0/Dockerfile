# syntax=docker/dockerfile:1.4
FROM gradle:jdk17-focal AS build
WORKDIR /app
RUN git clone --branch v0.1.0 https://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@github.com/bufbuild/connect-kotlin.git
WORKDIR /app/connect-kotlin
RUN gradle --version -g $(pwd)
RUN gradle protoc-gen-connect-kotlin:jar --no-daemon -g $(pwd)

FROM openjdk:8-jre-slim
COPY --from=build /app /app
WORKDIR /app/connect-kotlin
ENTRYPOINT ["exec", "java", "-jar", "protoc-gen-connect-kotlin/build/libs/protoc-gen-connect-kotlin.jar"]

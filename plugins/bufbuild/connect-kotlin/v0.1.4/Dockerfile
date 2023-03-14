# syntax=docker/dockerfile:1.4
FROM gradle:7.4.0-jdk17 AS build
WORKDIR /app
RUN git clone --depth 1 --branch v0.1.4 https://github.com/bufbuild/connect-kotlin.git
WORKDIR /app/connect-kotlin
RUN gradle --no-daemon -g $(pwd) protoc-gen-connect-kotlin:jar

FROM gcr.io/distroless/java17-debian11
WORKDIR /app
COPY --from=build /app/connect-kotlin/protoc-gen-connect-kotlin/build/libs/protoc-gen-connect-kotlin.jar /app
CMD ["/app/protoc-gen-connect-kotlin.jar"]

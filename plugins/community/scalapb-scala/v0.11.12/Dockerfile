# syntax=docker/dockerfile:1.4
FROM debian:bullseye-20221114 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl

RUN curl -fsSL -o protoc-gen-scala.sh https://repo1.maven.org/maven2/com/thesamet/scalapb/protoc-gen-scala/0.11.12/protoc-gen-scala-0.11.12-unix.sh \
    && chmod +x protoc-gen-scala.sh

FROM gcr.io/distroless/java11-debian11
COPY --from=build /bin/sh /bin/sh
COPY --from=build --link /protoc-gen-scala.sh .
USER nobody
ENTRYPOINT [ "/bin/sh", "/protoc-gen-scala.sh" ]

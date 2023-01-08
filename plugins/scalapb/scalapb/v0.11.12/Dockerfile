#  syntax=docker/dockerfile:1.4
FROM debian:bullseye-20221114 AS build

ARG TARGETARCH

WORKDIR /build
RUN apt-get update \
 && apt-get install -y curl \ 
 && apt-get install -y zip
# no support for arm atm, not sure if that is a big deal or not
RUN curl -fsSL -o scalapbc.zip https://github.com/scalapb/ScalaPB/releases/download/v0.11.11/protoc-gen-scala-0.11.11-linux-x86_64.zip \
    && unzip scalapbc

FROM gcr.io/distroless/base-debian11
COPY --from=build --link /build/protoc-gen-scala .
USER nobody
ENTRYPOINT [ "/protoc-gen-scala" ]

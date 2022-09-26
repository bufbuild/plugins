# syntax=docker/dockerfile:1.4
FROM debian:bullseye-20220912 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl \
 && if [ "${TARGETARCH}" = "arm64" ]; then TARGETARCH=aarch64; fi \
 && curl -fsSL https://github.com/grpc/grpc-web/releases/download/1.4.0/protoc-gen-grpc-web-1.4.0-linux-${TARGETARCH} -o /tmp/protoc-gen-grpc-web

FROM debian:bullseye-20220912-slim
COPY --from=build --link --chmod=0755 --chown=root:root /tmp/protoc-gen-grpc-web .
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-web" ]

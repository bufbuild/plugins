# syntax=docker/dockerfile:1.4
FROM debian:bullseye-20221024-slim AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl \
 && if [ "${TARGETARCH}" = "arm64" ]; then TARGETARCH=aarch64; else TARGETARCH=x86_64; fi \
 && curl -fsSL https://github.com/grpc/grpc-web/releases/download/1.4.2/protoc-gen-grpc-web-1.4.2-linux-${TARGETARCH} -o /tmp/protoc-gen-grpc-web

FROM debian:bullseye-20221024-slim
COPY --from=build --link --chmod=0755 --chown=root:root /tmp/protoc-gen-grpc-web .
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-web" ]

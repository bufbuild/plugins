# syntax=docker/dockerfile:1.4
FROM debian:bullseye-20230904 AS build

ARG TARGETARCH

WORKDIR /build
RUN apt-get update \
 && apt-get install -y curl
RUN arch=${TARGETARCH}; \
    if [ "${arch}" = "arm64" ]; then\
        arch="aarch_64"; \
    elif [ "${arch}" = "amd64" ]; then\
        arch="x86_64"; \
    fi; \
    echo "${arch}"; \
    curl -fsSL -o protoc-gen-reactor-grpc https://repo1.maven.org/maven2/com/salesforce/servicelibs/reactor-grpc/1.2.4/reactor-grpc-1.2.4-linux-${arch}.exe

FROM gcr.io/distroless/java17-debian11
COPY --from=build --link --chmod=0755 --chown=root:root /build/protoc-gen-reactor-grpc .
USER nobody
ENTRYPOINT [ "/protoc-gen-reactor-grpc" ]

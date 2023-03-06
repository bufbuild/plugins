# syntax=docker/dockerfile:1.4
FROM rust:1.67.1-alpine3.17 as builder
RUN apk add --no-cache musl-dev
WORKDIR /app
RUN --mount=type=cache,target=/usr/local/cargo/registry --mount=type=cache,target=/root/target \
    cargo install protoc-gen-tonic --version 0.2.2 --locked --root /app

FROM gcr.io/distroless/static
COPY --from=builder /app/bin/protoc-gen-tonic /protoc-gen-tonic
USER nobody
ENTRYPOINT ["/protoc-gen-tonic"]

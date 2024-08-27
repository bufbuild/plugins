# syntax=docker/dockerfile:1.9
FROM rust:1.80.1-alpine3.20 AS builder
RUN apk add --no-cache musl-dev
WORKDIR /app
ENV CARGO_REGISTRIES_CRATES_IO_PROTOCOL=sparse
RUN --mount=type=cache,target=/usr/local/cargo/registry,sharing=locked --mount=type=cache,target=/root/target \
    cargo install protoc-gen-tonic --version 0.4.1 --locked --root /app

FROM gcr.io/distroless/static-debian12:latest@sha256:ce46866b3a5170db3b49364900fb3168dc0833dfb46c26da5c77f22abb01d8c3 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=builder /app/bin/protoc-gen-tonic /protoc-gen-tonic
USER nobody
ENTRYPOINT ["/protoc-gen-tonic"]

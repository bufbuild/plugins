# syntax=docker/dockerfile:1.24
FROM rust:1.96.0-alpine3.23@sha256:66f48b19d6e88519e2e58bebe0d945779a6a4ca41c2db17db78c9569655b50ac AS builder
RUN apk add --no-cache musl-dev
WORKDIR /app
ENV CARGO_REGISTRIES_CRATES_IO_PROTOCOL=sparse
# The protoc-gen-connect-rust binary is a [[bin]] target in the
# connectrpc-codegen crate. cargo install compiles it by crate name.
RUN --mount=type=cache,target=/usr/local/cargo/registry,sharing=locked --mount=type=cache,target=/root/target \
    cargo install connectrpc-codegen --version 0.7.0 --locked --root /app

FROM gcr.io/distroless/static-debian13:latest@sha256:3592aa8171c77482f62bbc4164e6a2d141c6122554ace66e5cc910cadb961ff0 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=builder /app/bin/protoc-gen-connect-rust /protoc-gen-connect-rust
USER nobody
ENTRYPOINT ["/protoc-gen-connect-rust"]

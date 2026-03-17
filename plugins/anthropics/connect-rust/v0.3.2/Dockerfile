# syntax=docker/dockerfile:1.19
FROM rust:1.91.1-alpine3.22 AS builder
RUN apk add --no-cache musl-dev
WORKDIR /app
ENV CARGO_REGISTRIES_CRATES_IO_PROTOCOL=sparse
# The protoc-gen-connect-rust binary is a [[bin]] target in the
# connectrpc-codegen crate. cargo install compiles it by crate name.
RUN --mount=type=cache,target=/usr/local/cargo/registry,sharing=locked --mount=type=cache,target=/root/target \
    cargo install connectrpc-codegen --version 0.3.2 --locked --root /app

FROM gcr.io/distroless/static-debian12:latest@sha256:87bce11be0af225e4ca761c40babb06d6d559f5767fbf7dc3c47f0f1a466b92c AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=builder /app/bin/protoc-gen-connect-rust /protoc-gen-connect-rust
USER nobody
ENTRYPOINT ["/protoc-gen-connect-rust"]

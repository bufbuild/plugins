# syntax=docker/dockerfile:1.24
FROM rust:1.96.1-alpine3.24@sha256:a41f7740f8b45d45795624eec13a8b42263cc700f19f7e4e86e04d3dda08a479 AS builder
RUN apk add --no-cache musl-dev
WORKDIR /app
ENV CARGO_REGISTRIES_CRATES_IO_PROTOCOL=sparse
# The protoc-gen-connect-rust binary is a [[bin]] target in the
# connectrpc-codegen crate. cargo install compiles it by crate name.
RUN --mount=type=cache,target=/usr/local/cargo/registry,sharing=locked --mount=type=cache,target=/root/target \
    cargo install connectrpc-codegen --version 0.8.0 --locked --root /app

FROM gcr.io/distroless/static-debian13:latest@sha256:3592aa8171c77482f62bbc4164e6a2d141c6122554ace66e5cc910cadb961ff0 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=builder /app/bin/protoc-gen-connect-rust /protoc-gen-connect-rust
USER nobody
ENTRYPOINT ["/protoc-gen-connect-rust"]

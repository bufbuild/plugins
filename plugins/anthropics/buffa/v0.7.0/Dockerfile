# syntax=docker/dockerfile:1.24
FROM rust:1.95.0-alpine3.23@sha256:606fd313a0f49743ee2a7bd49a0914bab7deedb12791f3a846a34a4711db7ed2 AS builder
RUN apk add --no-cache musl-dev
WORKDIR /app
ENV CARGO_REGISTRIES_CRATES_IO_PROTOCOL=sparse
RUN --mount=type=cache,target=/usr/local/cargo/registry,sharing=locked --mount=type=cache,target=/root/target \
    cargo install protoc-gen-buffa --version 0.7.0 --locked --root /app

FROM gcr.io/distroless/static-debian13:latest@sha256:3592aa8171c77482f62bbc4164e6a2d141c6122554ace66e5cc910cadb961ff0 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=builder /app/bin/protoc-gen-buffa /protoc-gen-buffa
USER nobody
ENTRYPOINT ["/protoc-gen-buffa"]

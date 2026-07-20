# syntax=docker/dockerfile:1.24
FROM rust:1.97.0-alpine3.24@sha256:ec9c91e77119ce498cd1e87d96d77e0f75b2cee21655a29bc2bf75a51a2b20a4 AS builder
RUN apk add --no-cache musl-dev
WORKDIR /app
ENV CARGO_REGISTRIES_CRATES_IO_PROTOCOL=sparse
RUN --mount=type=cache,target=/usr/local/cargo/registry,sharing=locked --mount=type=cache,target=/root/target \
    cargo install protoc-gen-buffa --version 0.9.0 --locked --root /app

FROM gcr.io/distroless/static-debian13:latest@sha256:9197324ba51d9cd071af8505989365c006adf9d6d2067eada25aef00abbb5278 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=builder /app/bin/protoc-gen-buffa /protoc-gen-buffa
USER nobody
ENTRYPOINT ["/protoc-gen-buffa"]

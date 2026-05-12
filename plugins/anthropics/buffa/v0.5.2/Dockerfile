# syntax=docker/dockerfile:1.23
FROM rust:1.95.0-alpine3.23@sha256:606fd313a0f49743ee2a7bd49a0914bab7deedb12791f3a846a34a4711db7ed2 AS builder
RUN apk add --no-cache musl-dev
WORKDIR /app
ENV CARGO_REGISTRIES_CRATES_IO_PROTOCOL=sparse
RUN --mount=type=cache,target=/usr/local/cargo/registry,sharing=locked --mount=type=cache,target=/root/target \
    cargo install protoc-gen-buffa --version 0.5.2 --locked --root /app

FROM gcr.io/distroless/static-debian13:latest@sha256:47b2d72ff90843eb8a768b5c2f89b40741843b639d065b9b937b07cd59b479c6 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=builder /app/bin/protoc-gen-buffa /protoc-gen-buffa
USER nobody
ENTRYPOINT ["/protoc-gen-buffa"]

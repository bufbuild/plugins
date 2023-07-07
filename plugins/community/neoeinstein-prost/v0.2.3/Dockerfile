# syntax=docker/dockerfile:1.4
FROM rust:1.70.0-alpine3.18 as builder
RUN apk add --no-cache musl-dev
WORKDIR /app
ENV CARGO_REGISTRIES_CRATES_IO_PROTOCOL=sparse
RUN --mount=type=cache,target=/usr/local/cargo/registry --mount=type=cache,target=/root/target \
    cargo install protoc-gen-prost --version 0.2.3 --locked --root /app

FROM gcr.io/distroless/static
COPY --from=builder /app/bin/protoc-gen-prost /protoc-gen-prost
USER nobody
ENTRYPOINT ["/protoc-gen-prost"]

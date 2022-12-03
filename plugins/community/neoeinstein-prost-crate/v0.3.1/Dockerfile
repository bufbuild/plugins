# syntax=docker/dockerfile:1.4
FROM rust:1.65.0-alpine3.16 as builder
RUN apk add --no-cache musl-dev
WORKDIR /app
RUN cargo install protoc-gen-prost-crate --version 0.3.1 --locked --root /app

FROM gcr.io/distroless/cc
COPY --from=builder /app/bin/protoc-gen-prost-crate /protoc-gen-prost-crate
USER nobody
ENTRYPOINT ["/protoc-gen-prost-crate"]

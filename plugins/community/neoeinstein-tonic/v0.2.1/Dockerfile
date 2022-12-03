# syntax=docker/dockerfile:1.4
FROM rust:1.65.0-alpine3.16 as builder
RUN apk add --no-cache musl-dev
WORKDIR /app
RUN cargo install protoc-gen-tonic --version 0.2.1 --locked --root /app

FROM gcr.io/distroless/cc
COPY --from=builder /app/bin/protoc-gen-tonic /protoc-gen-tonic
USER nobody
ENTRYPOINT ["/protoc-gen-tonic"]

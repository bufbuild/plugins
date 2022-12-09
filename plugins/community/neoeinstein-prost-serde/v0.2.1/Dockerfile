# syntax=docker/dockerfile:1.4
FROM rust:1.65.0-alpine3.16 as builder
RUN apk add --no-cache musl-dev
WORKDIR /app
RUN cargo install protoc-gen-prost-serde --version 0.2.1 --locked --root /app

FROM gcr.io/distroless/static
COPY --from=builder /app/bin/protoc-gen-prost-serde /protoc-gen-prost-serde
USER nobody
ENTRYPOINT ["/protoc-gen-prost-serde"]

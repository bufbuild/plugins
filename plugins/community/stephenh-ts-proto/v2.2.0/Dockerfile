# syntax=docker/dockerfile:1.10
FROM node:22.9.0-bookworm AS build
WORKDIR /app
COPY --link package*.json .
RUN npm ci
RUN sed -i -e 's|/usr/bin/env node|/nodejs/bin/node|g' /app/node_modules/ts-proto/protoc-gen-ts_proto \
 && rm -f /app/node_modules/dprint-node/*darwin*.node /app/node_modules/dprint-node/*win32*.node

FROM gcr.io/distroless/nodejs22-debian12:latest@sha256:ae1ca595a02007aaeeffcb0c3558c909fb78e04028f0206b864d2281e7927442 AS node

FROM gcr.io/distroless/cc-debian12:latest@sha256:b6e1e913f633495eeb80a41e03de1a41aa863e9b19902309b180ffdc4b99db2c AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=node --chmod=0755 /nodejs/bin/node /nodejs/bin/node
COPY --link --from=build /app /app
USER nobody
ENTRYPOINT ["/nodejs/bin/node"]
CMD [ "/app/node_modules/.bin/protoc-gen-ts_proto" ]

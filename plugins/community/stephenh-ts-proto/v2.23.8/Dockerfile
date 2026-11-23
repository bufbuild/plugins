# syntax=docker/dockerfile:1.29
FROM node:26.13.1-trixie@sha256:e18c3ccdf7d71619990b0e85639f506e2028a95410999cab24cc373cbd10e0ac AS build
WORKDIR /app
COPY --link package*.json .
RUN npm ci
RUN sed -i -e 's|/usr/bin/env node|/nodejs/bin/node|g' /app/node_modules/ts-proto/protoc-gen-ts_proto \
 && rm -f /app/node_modules/dprint-node/*darwin*.node /app/node_modules/dprint-node/*win32*.node

FROM gcr.io/distroless/nodejs26-debian13:latest@sha256:80631bd2f0e7ef192138a7409013b47d92fd11b8ff3bb35f5c0a65be3ecaea4c AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build /app /app
USER nobody
ENTRYPOINT ["/nodejs/bin/node"]
CMD [ "/app/node_modules/.bin/protoc-gen-ts_proto" ]

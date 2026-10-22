# syntax=docker/dockerfile:1.27
FROM node:26.12.0-trixie@sha256:1b84b89ada025e3b14543cbb837ffaa1c1b5273b841c5b94dd1a8f9c1d8cf7fe AS build
WORKDIR /app
COPY --link package*.json .
RUN npm ci
RUN sed -i -e 's|/usr/bin/env node|/nodejs/bin/node|g' /app/node_modules/ts-proto/protoc-gen-ts_proto \
 && rm -f /app/node_modules/dprint-node/*darwin*.node /app/node_modules/dprint-node/*win32*.node

FROM gcr.io/distroless/nodejs26-debian13:latest@sha256:456bd22a48bc4b8b82b19041d7d9ff9c14e4ff0b80ea32c0b912be2f8288dbf7 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build /app /app
USER nobody
ENTRYPOINT ["/nodejs/bin/node"]
CMD [ "/app/node_modules/.bin/protoc-gen-ts_proto" ]

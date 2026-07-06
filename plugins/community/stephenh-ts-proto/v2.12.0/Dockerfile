# syntax=docker/dockerfile:1.24
FROM node:24.18.0-trixie@sha256:5ac1f09047e215d1e4df7fbdf95ff86ca14a35632380b6672d6c9c62d5ac3800 AS build
WORKDIR /app
COPY --link package*.json .
RUN npm ci
RUN sed -i -e 's|/usr/bin/env node|/nodejs/bin/node|g' /app/node_modules/ts-proto/protoc-gen-ts_proto \
 && rm -f /app/node_modules/dprint-node/*darwin*.node /app/node_modules/dprint-node/*win32*.node

FROM gcr.io/distroless/nodejs24-debian13:latest@sha256:ef5f3caf80da1630edd1a4df7b307a8f7d4553f8eec1dd29852b76e793593903 AS node

FROM gcr.io/distroless/cc-debian13:latest@sha256:a017e74bd2a12d98342dbecd33d121d2b160415ed777573dc1808969e989d94d AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=node --chmod=0755 /nodejs/bin/node /nodejs/bin/node
COPY --link --from=build /app /app
USER nobody
ENTRYPOINT ["/nodejs/bin/node"]
CMD [ "/app/node_modules/.bin/protoc-gen-ts_proto" ]

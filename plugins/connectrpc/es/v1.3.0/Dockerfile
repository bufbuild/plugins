# syntax=docker/dockerfile:1.6
FROM node:20.10.0-alpine3.18 AS build
WORKDIR /app
COPY --link package*.json .
RUN npm ci
RUN sed -i -e 's|/usr/bin/env node|/nodejs/bin/node|g' /app/node_modules/@connectrpc/protoc-gen-connect-es/bin/protoc-gen-connect-es

FROM gcr.io/distroless/nodejs20-debian12:latest@sha256:7715474a901a28e3edcdf7730f14b33e30c26085989ce04b0de163fe8fab0f03
COPY --link --from=build /app /app
USER nobody
ENTRYPOINT [ "/app/node_modules/.bin/protoc-gen-connect-es" ]

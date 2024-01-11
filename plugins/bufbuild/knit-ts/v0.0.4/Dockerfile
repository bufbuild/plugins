# syntax=docker/dockerfile:1.4
FROM node:20.11.0-alpine3.19 AS build
WORKDIR /app
COPY --link package*.json .
RUN npm ci
RUN sed -i -e 's|/usr/bin/env node|/nodejs/bin/node|g' /app/node_modules/@bufbuild/protoc-gen-knit-ts/bin/protoc-gen-knit-ts

FROM gcr.io/distroless/nodejs20-debian12:latest@sha256:7715474a901a28e3edcdf7730f14b33e30c26085989ce04b0de163fe8fab0f03
COPY --link --from=build /app /app
USER nobody
ENTRYPOINT [ "/app/node_modules/.bin/protoc-gen-knit-ts" ]

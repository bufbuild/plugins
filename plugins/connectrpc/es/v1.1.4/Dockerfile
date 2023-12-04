# syntax=docker/dockerfile:1.6
FROM node:20.10.0-alpine3.18 AS build
WORKDIR /app
COPY --link package*.json .
RUN npm ci
RUN sed -i -e 's|/usr/bin/env node|/nodejs/bin/node|g' /app/node_modules/@connectrpc/protoc-gen-connect-es/bin/protoc-gen-connect-es

FROM gcr.io/distroless/nodejs20-debian12:latest@sha256:6bd06494907b362c5d85dfbf60930e98272b9ffc1cbcb5c6d602da7ba9a303a0
COPY --link --from=build /app /app
USER nobody
ENTRYPOINT [ "/app/node_modules/.bin/protoc-gen-connect-es" ]

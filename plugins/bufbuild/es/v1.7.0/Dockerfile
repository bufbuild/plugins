# syntax=docker/dockerfile:1.4
FROM node:20.11.0-alpine3.19 AS build
WORKDIR /app
COPY --link package*.json .
RUN npm ci
RUN sed -i -e 's|/usr/bin/env node|/nodejs/bin/node|g' /app/node_modules/@bufbuild/protoc-gen-es/bin/protoc-gen-es

FROM gcr.io/distroless/nodejs20-debian12:latest@sha256:c63fbd00d837baee8160c764124f9f33a9bea2b38e0c4d6306453e43ac6bf443
COPY --link --from=build /app /app
USER nobody
ENTRYPOINT [ "/app/node_modules/.bin/protoc-gen-es" ]

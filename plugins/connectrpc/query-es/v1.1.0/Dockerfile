# syntax=docker/dockerfile:1.4
FROM node:20.10.0-alpine3.18 AS build
WORKDIR /app
COPY --link package*.json .
RUN npm ci
RUN sed -i -e 's|/usr/bin/env node|/nodejs/bin/node|g' /app/node_modules/@connectrpc/protoc-gen-connect-query/bin/protoc-gen-connect-query

FROM gcr.io/distroless/nodejs20-debian12:latest@sha256:3f46c2fa879281ea5969d0aedd52ba1d82ed7687d0b68df850d2056564285efd
COPY --link --from=build /app /app
USER nobody
ENTRYPOINT [ "/app/node_modules/.bin/protoc-gen-connect-query" ]

# syntax=docker/dockerfile:1.4
FROM node:18.15.0-alpine3.17 AS build
WORKDIR /app
COPY --link package*.json .
RUN npm ci
RUN sed -i -e 's|/usr/bin/env node|/nodejs/bin/node|g' /app/node_modules/@bufbuild/protoc-gen-connect-query/bin/protoc-gen-connect-query

FROM gcr.io/distroless/nodejs18-debian11
COPY --link --from=build /app /app
USER nobody
ENTRYPOINT [ "/app/node_modules/.bin/protoc-gen-connect-query" ]

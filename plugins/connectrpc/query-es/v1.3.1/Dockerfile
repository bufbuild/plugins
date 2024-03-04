# syntax=docker/dockerfile:1.4
FROM node:20.11.1-alpine3.19 AS build
WORKDIR /app
COPY --link package*.json .
RUN npm ci
RUN sed -i -e 's|/usr/bin/env node|/nodejs/bin/node|g' /app/node_modules/@connectrpc/protoc-gen-connect-query/bin/protoc-gen-connect-query

FROM gcr.io/distroless/nodejs20-debian12:latest@sha256:a827f8a950b8b393862de98f1b2b37a96558bd33dde8ab8bdeac88f35cbead2e
COPY --link --from=build /app /app
USER nobody
ENTRYPOINT [ "/app/node_modules/.bin/protoc-gen-connect-query" ]

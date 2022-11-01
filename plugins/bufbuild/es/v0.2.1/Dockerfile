# syntax=docker/dockerfile:1.4
FROM node:16.18.0-bullseye-slim AS build
WORKDIR /app
COPY --link package*.json .
RUN npm ci

FROM node:16.18.0-bullseye-slim
COPY --link --from=build /app /app
USER nobody
ENTRYPOINT [ "/app/node_modules/.bin/protoc-gen-es" ]

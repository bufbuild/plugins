# syntax=docker/dockerfile:1.4
FROM node:17.9.1-bullseye-slim
ARG PLUGIN_VERSION
RUN test -n "${PLUGIN_VERSION}"
WORKDIR /app
COPY --link package*.json .
RUN npm ci

USER nobody
ENTRYPOINT [ "/app/node_modules/.bin/protoc-gen-connect-web" ]

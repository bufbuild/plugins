# syntax=docker/dockerfile:1.4
FROM node:lts-bullseye-slim
ARG PLUGIN_VERSION
RUN test -n "${PLUGIN_VERSION}"
WORKDIR /app
RUN npm install @bufbuild/protoc-gen-es@${PLUGIN_VERSION}

USER nobody
ENTRYPOINT [ "/app/node_modules/.bin/protoc-gen-es" ]

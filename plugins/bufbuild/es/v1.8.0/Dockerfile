# syntax=docker/dockerfile:1.4
FROM node:20.11.1-alpine3.19 AS build
WORKDIR /app
COPY --link package*.json .
RUN npm ci
RUN sed -i -e 's|/usr/bin/env node|/nodejs/bin/node|g' /app/node_modules/@bufbuild/protoc-gen-es/bin/protoc-gen-es

FROM gcr.io/distroless/nodejs20-debian12:latest@sha256:e0edb89411e3f93863e9bd97c15cddcbac968d7a16b7a82af6ae6b264da65de9
COPY --link --from=build /app /app
USER nobody
ENTRYPOINT [ "/app/node_modules/.bin/protoc-gen-es" ]

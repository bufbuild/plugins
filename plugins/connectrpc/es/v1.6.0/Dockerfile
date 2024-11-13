# syntax=docker/dockerfile:1.10
FROM node:22.9.0-bookworm AS build
WORKDIR /app
COPY --link package*.json .
RUN npm ci \
 && find node_modules/typescript ! -name 'typescript.js' ! -name 'package.json' -type f -exec rm -f {} + \
 && find node_modules/typescript -depth -type d -empty -delete \
 && ./node_modules/.bin/esbuild ./node_modules/.bin/protoc-gen-connect-es --bundle --external:typescript --platform=node --outfile=protoc-gen-connect-es.js

FROM gcr.io/distroless/nodejs22-debian12:latest@sha256:d7ccebdf7617f225aa511c4a0e9c3bff2a8a65b22f8032ca982193d5a52c8ee9 AS node

FROM gcr.io/distroless/cc-debian12:latest@sha256:3310655aac0d85eb9d579792387af1ff3eb7a1667823478be58020ab0e0d97a8 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=node --chmod=0755 /nodejs/bin/node /nodejs/bin/node
COPY --link --from=build --chmod=0755 /app/protoc-gen-connect-es.js /app/protoc-gen-connect-es.js
COPY --link --from=build /app/node_modules/typescript /app/node_modules/typescript
USER nobody
ENTRYPOINT ["/nodejs/bin/node"]
CMD [ "/app/protoc-gen-connect-es.js" ]

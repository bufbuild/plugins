# syntax=docker/dockerfile:1.19
FROM node:24.13.0-bookworm AS build
WORKDIR /app
COPY --link package*.json .
RUN npm ci \
 && find node_modules/typescript ! -name 'typescript.js' ! -name 'package.json' -type f -exec rm -f {} + \
 && find node_modules/typescript -depth -type d -empty -delete \
 && ./node_modules/.bin/esbuild ./node_modules/.bin/protoc-gen-es --bundle --external:typescript --platform=node --outfile=protoc-gen-es.js

FROM gcr.io/distroless/nodejs24-debian12:latest@sha256:aa9ea28c521590ff497a06449109f26582ca9fa7eacfe8b169fec24399c1bd8a AS node

FROM gcr.io/distroless/cc-debian12:latest@sha256:66d87e170bc2c5e2b8cf853501141c3c55b4e502b8677595c57534df54a68cc5 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=node --chmod=0755 /nodejs/bin/node /nodejs/bin/node
COPY --link --from=build --chmod=0755 /app/protoc-gen-es.js /app/protoc-gen-es.js
COPY --link --from=build /app/node_modules/typescript /app/node_modules/typescript
USER nobody
ENTRYPOINT ["/nodejs/bin/node"]
CMD [ "/app/protoc-gen-es.js" ]

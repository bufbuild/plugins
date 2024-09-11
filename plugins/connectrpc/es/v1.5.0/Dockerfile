# syntax=docker/dockerfile:1.10
FROM node:20.17.0-bookworm AS build
WORKDIR /app
COPY --link package*.json .
RUN npm ci \
 && find node_modules/typescript ! -name 'typescript.js' ! -name 'package.json' -type f -exec rm -f {} + \
 && find node_modules/typescript -depth -type d -empty -delete \
 && ./node_modules/.bin/esbuild ./node_modules/.bin/protoc-gen-connect-es --bundle --external:typescript --platform=node --outfile=protoc-gen-connect-es.js

FROM gcr.io/distroless/nodejs20-debian12:latest@sha256:bdbbd3d1bb68ab13bcb075a8f38973acf2129892ac3daafbe96e0b6c66681296 AS node

FROM gcr.io/distroless/cc-debian12:latest@sha256:b6e1e913f633495eeb80a41e03de1a41aa863e9b19902309b180ffdc4b99db2c AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=node --chmod=0755 /nodejs/bin/node /nodejs/bin/node
COPY --link --from=build --chmod=0755 /app/protoc-gen-connect-es.js /app/protoc-gen-connect-es.js
COPY --link --from=build /app/node_modules/typescript /app/node_modules/typescript
USER nobody
ENTRYPOINT ["/nodejs/bin/node"]
CMD [ "/app/protoc-gen-connect-es.js" ]

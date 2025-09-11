# syntax=docker/dockerfile:1.17
FROM node:22.19.0-bookworm AS build
WORKDIR /app
COPY --link package*.json .
RUN npm ci \
 && find node_modules/typescript ! -name 'typescript.js' ! -name 'package.json' -type f -exec rm -f {} + \
 && find node_modules/typescript -depth -type d -empty -delete \
 && ./node_modules/.bin/esbuild ./node_modules/.bin/protoc-gen-es --bundle --external:typescript --platform=node --outfile=protoc-gen-es.js

FROM gcr.io/distroless/nodejs22-debian12:latest@sha256:3732180ba4a39101bd95b7105ef0c47526c197d6c29c6d48f7059a647a4064a5 AS node

FROM gcr.io/distroless/cc-debian12:latest@sha256:620d8b11ae800f0dbd7995f89ddc5344ad603269ea98770588b1b07a4a0a6872 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=node --chmod=0755 /nodejs/bin/node /nodejs/bin/node
COPY --link --from=build --chmod=0755 /app/protoc-gen-es.js /app/protoc-gen-es.js
COPY --link --from=build /app/node_modules/typescript /app/node_modules/typescript
USER nobody
ENTRYPOINT ["/nodejs/bin/node"]
CMD [ "/app/protoc-gen-es.js" ]

# syntax=docker/dockerfile:1.10
FROM node:22.11.0-bookworm AS build
WORKDIR /app
COPY --link package*.json .
RUN npm ci \
 && find node_modules/typescript ! -name 'typescript.js' ! -name 'package.json' -type f -exec rm -f {} + \
 && find node_modules/typescript -depth -type d -empty -delete \
 && ./node_modules/.bin/esbuild ./node_modules/.bin/protoc-gen-connect-query --bundle --external:typescript --platform=node --outfile=protoc-gen-connect-query.js

FROM gcr.io/distroless/nodejs22-debian12:latest@sha256:c218f62198d07fc67e36fff5639985f29b1bdcf04a601c1d23c0ab1121f55f0b AS node

FROM gcr.io/distroless/cc-debian12:latest@sha256:2fb69596e692931f909c4c69ab09e50608959eaf8898c44fa64db741a23588b0 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=node --chmod=0755 /nodejs/bin/node /nodejs/bin/node
COPY --link --from=build --chmod=0755 /app/protoc-gen-connect-query.js /app/protoc-gen-connect-query.js
COPY --link --from=build /app/node_modules/typescript /app/node_modules/typescript
USER nobody
ENTRYPOINT ["/nodejs/bin/node"]
CMD [ "/app/protoc-gen-connect-query.js" ]

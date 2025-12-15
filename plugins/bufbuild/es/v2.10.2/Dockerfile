# syntax=docker/dockerfile:1.19
FROM node:24.12.0-bookworm AS build
WORKDIR /app
COPY --link package*.json .
RUN npm ci \
 && find node_modules/typescript ! -name 'typescript.js' ! -name 'package.json' -type f -exec rm -f {} + \
 && find node_modules/typescript -depth -type d -empty -delete \
 && ./node_modules/.bin/esbuild ./node_modules/.bin/protoc-gen-es --bundle --external:typescript --platform=node --outfile=protoc-gen-es.js

FROM gcr.io/distroless/nodejs24-debian12:latest@sha256:cdf42f994cee425970f30a0d0e37112dcc800737dde7bba50415612702b399af AS node

FROM gcr.io/distroless/cc-debian12:latest@sha256:0c8eac8ea42a167255d03c3ba6dfad2989c15427ed93d16c53ef9706ea4691df AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=node --chmod=0755 /nodejs/bin/node /nodejs/bin/node
COPY --link --from=build --chmod=0755 /app/protoc-gen-es.js /app/protoc-gen-es.js
COPY --link --from=build /app/node_modules/typescript /app/node_modules/typescript
USER nobody
ENTRYPOINT ["/nodejs/bin/node"]
CMD [ "/app/protoc-gen-es.js" ]

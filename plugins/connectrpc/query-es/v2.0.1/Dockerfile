# syntax=docker/dockerfile:1.12
FROM node:22.12.0-bookworm AS build
WORKDIR /app
COPY --link package*.json .
RUN npm ci \
 && find node_modules/typescript ! -name 'typescript.js' ! -name 'package.json' -type f -exec rm -f {} + \
 && find node_modules/typescript -depth -type d -empty -delete \
 && ./node_modules/.bin/esbuild ./node_modules/.bin/protoc-gen-connect-query --bundle --external:typescript --platform=node --outfile=protoc-gen-connect-query.js

FROM gcr.io/distroless/nodejs22-debian12:latest@sha256:b2811d7945a1f3ef91cf8a2ead779171b438eb0011f3bdee0c8cc60cb90b2c0c AS node

FROM gcr.io/distroless/cc-debian12:latest@sha256:f913198471738d9eedcd00c0ca812bf663e8959eebff3a3cbadb027ed9da0c38 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=node --chmod=0755 /nodejs/bin/node /nodejs/bin/node
COPY --link --from=build --chmod=0755 /app/protoc-gen-connect-query.js /app/protoc-gen-connect-query.js
COPY --link --from=build /app/node_modules/typescript /app/node_modules/typescript
USER nobody
ENTRYPOINT ["/nodejs/bin/node"]
CMD [ "/app/protoc-gen-connect-query.js" ]

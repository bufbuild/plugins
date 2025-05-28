# syntax=docker/dockerfile:1.15
FROM node:22.16.0-bookworm AS build
WORKDIR /app
COPY --link package*.json .
RUN npm ci \
 && ./node_modules/.bin/esbuild ./node_modules/.bin/protoc-gen-ts --bundle --external:typescript --platform=node --outfile=protoc-gen-ts.js

FROM gcr.io/distroless/nodejs22-debian12:latest@sha256:d028bfd3111bb0e2a75ef5e2232fa91cb826f9121a66a2242962b1c52398a237 AS node

FROM gcr.io/distroless/cc-debian12:latest@sha256:c53c9416a1acdbfd6e09abba720442444a3d1a6338b8db850e5e198b59af5570 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=node --chmod=0755 /nodejs/bin/node /nodejs/bin/node
COPY --link --from=build --chmod=0755 /app/protoc-gen-ts.js /app/protoc-gen-ts.js
COPY --link --from=build /app/node_modules/typescript /app/node_modules/typescript
USER nobody
ENTRYPOINT ["/nodejs/bin/node"]
CMD [ "/app/protoc-gen-ts.js" ]

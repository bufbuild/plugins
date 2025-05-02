# syntax=docker/dockerfile:1.12
FROM node:22.15.0-bookworm AS build
WORKDIR /app
COPY --link package*.json .
RUN npm ci \
 && ./node_modules/.bin/esbuild ./node_modules/.bin/protoc-gen-ts --bundle --external:typescript --platform=node --outfile=protoc-gen-ts.js

FROM gcr.io/distroless/nodejs22-debian12:latest@sha256:5bbfaef4976723a9574efdeea941ca4f2a30b271a8b9ad6a1036dbaae68f855d AS node

FROM gcr.io/distroless/cc-debian12:latest@sha256:c1cbcec08d39c81adbefb80cabc51cba285465866f7b5ab15ddb2fcae51a1aed AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=node --chmod=0755 /nodejs/bin/node /nodejs/bin/node
COPY --link --from=build --chmod=0755 /app/protoc-gen-ts.js /app/protoc-gen-ts.js
COPY --link --from=build /app/node_modules/typescript /app/node_modules/typescript
USER nobody
ENTRYPOINT ["/nodejs/bin/node"]
CMD [ "/app/protoc-gen-ts.js" ]

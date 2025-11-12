# syntax=docker/dockerfile:1.19
FROM node:24.11.0-bookworm AS build
WORKDIR /app
COPY --link package*.json .
RUN npm ci \
 && find node_modules/typescript ! -name 'typescript.js' ! -name 'package.json' -type f -exec rm -f {} + \
 && find node_modules/typescript -depth -type d -empty -delete \
 && ./node_modules/.bin/esbuild ./node_modules/.bin/protoc-gen-es --bundle --external:typescript --platform=node --outfile=protoc-gen-es.js

FROM gcr.io/distroless/nodejs24-debian12:latest@sha256:ac05a8febf24b547f09068149fcd6a071f69629cb2148b6f707a157e7f4c6306 AS node

FROM gcr.io/distroless/cc-debian12:latest@sha256:0000f9dc0290f8eaf0ecceafbc35e803649087ea7879570fbc78372df7ac649b AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=node --chmod=0755 /nodejs/bin/node /nodejs/bin/node
COPY --link --from=build --chmod=0755 /app/protoc-gen-es.js /app/protoc-gen-es.js
COPY --link --from=build /app/node_modules/typescript /app/node_modules/typescript
USER nobody
ENTRYPOINT ["/nodejs/bin/node"]
CMD [ "/app/protoc-gen-es.js" ]

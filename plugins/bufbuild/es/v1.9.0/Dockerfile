# syntax=docker/dockerfile:1.7
FROM node:20.12.2-alpine3.19 AS build
WORKDIR /app
COPY --link package*.json .
RUN npm ci \
 && find node_modules/typescript ! -name 'typescript.js' ! -name 'package.json' -type f -exec rm -f {} + \
 && find node_modules/typescript -depth -type d -empty -delete \
 && ./node_modules/.bin/esbuild ./node_modules/.bin/protoc-gen-es --bundle --external:typescript --platform=node --outfile=protoc-gen-es.js

FROM gcr.io/distroless/nodejs20-debian12:latest@sha256:6d0f50681aab846e8c005ee864324ef2cbea464fd10a09e3b02876cdc0696543 AS node

FROM gcr.io/distroless/cc-debian12:latest@sha256:7a01d633f75120af59c71489e0911fa8b6512673a3ff0b999522b4221ab4d86a
COPY --link --from=node --chmod=0755 /nodejs/bin/node /nodejs/bin/node
COPY --link --from=build --chmod=0755 /app/protoc-gen-es.js /app/protoc-gen-es.js
COPY --link --from=build /app/node_modules/typescript /app/node_modules/typescript
USER nobody
ENTRYPOINT ["/nodejs/bin/node"]
CMD [ "/app/protoc-gen-es.js" ]

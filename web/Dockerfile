# syntax=docker/dockerfile:1.7
# Multi-stage build for relay-web. Build context is `./web` (set in
# .github/workflows/publish-relay-web.yml) so paths are relative to web/.

FROM --platform=$BUILDPLATFORM node:24-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund

FROM --platform=$BUILDPLATFORM node:24-alpine AS build
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . ./
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

# Slim runtime — relies on `output: "standalone"` in next.config.mjs.
FROM node:24-alpine AS runtime
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3000
ENV HOSTNAME=0.0.0.0
RUN addgroup -S relay && adduser -S relay -G relay
COPY --from=build --chown=relay:relay /app/public ./public
COPY --from=build --chown=relay:relay /app/.next/standalone ./
COPY --from=build --chown=relay:relay /app/.next/static ./.next/static
USER relay
EXPOSE 3000
CMD ["node", "server.js"]

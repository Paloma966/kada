FROM node:22-alpine AS builder
WORKDIR /app

# The API base URL is inlined into the bundle at build time, so it must be a build argument.
# The default of "same-origin" keeps requests relative, which is what a container deployment wants:
# nginx serves this frontend and proxies /api/ to the Go backend. A deployment that talks to an API on
# another host overrides it with that URL. Leaving it unset would fall back to the development default
# (http://localhost:8080) in the browser, so it is never left unset on purpose.
ARG NEXT_PUBLIC_API_URL="same-origin"
ENV NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL}

COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:22-alpine
WORKDIR /app
ENV NODE_ENV=production HOSTNAME=0.0.0.0 PORT=3000
COPY --from=builder --chown=node:node /app/next.config.ts ./
COPY --from=builder --chown=node:node /app/public ./public
COPY --from=builder --chown=node:node /app/.next/standalone ./
COPY --from=builder --chown=node:node /app/.next/static ./.next/static
# Run as a non-root user (the node image ships the uid 1000 "node" user)
USER node
EXPOSE 3000
CMD ["node", "server.js"]

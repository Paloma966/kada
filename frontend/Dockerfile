FROM node:22-alpine AS builder
WORKDIR /app

# The API base URL is inlined into the bundle at build time, so it must be a build argument.
# The default is empty, which means "same origin": nginx serves this frontend and proxies /api/ to the
# Go backend. Setting it to a real URL is what a deployment would do to talk to an API on another host.
ARG NEXT_PUBLIC_API_URL=""
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

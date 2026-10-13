FROM node:24-alpine AS deps

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

FROM node:24-alpine AS build

WORKDIR /app

COPY --from=deps /app/node_modules ./node_modules
COPY . .

# API_ORIGIN is deliberately not set. Next resolves rewrites at build time, and
# in production there are none: the browser calls /api and /auth on this same
# hostname and the reverse proxy routes them to the api container. The rewrite
# exists only so `next dev` can reach an api running on another port.
ENV NEXT_TELEMETRY_DISABLED=1

RUN npm run build

FROM node:24-alpine AS run

WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3000
ENV HOSTNAME=0.0.0.0

# The standalone output carries only the dependencies actually reached, so the
# full node_modules tree is not copied forward.
COPY --from=build /app/.next/standalone ./
COPY --from=build /app/.next/static ./.next/static
COPY --from=build /app/public ./public

USER node

EXPOSE 3000

CMD ["node", "server.js"]

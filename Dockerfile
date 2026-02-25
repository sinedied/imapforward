# Build stage
FROM node:24-alpine AS build
WORKDIR /app
COPY package*.json tsconfig.json ./
COPY src/ ./src/
RUN npm ci && npm run build

# Production stage
FROM node:24-alpine
WORKDIR /app
COPY package*.json ./
RUN addgroup -g 1001 -S appgroup && \
    adduser -S appuser -u 1001 -G appgroup && \
    npm ci --omit=dev && \
    npm cache clean --force
COPY --from=build /app/dist ./dist
USER appuser

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["node", "dist/cli.js"]
CMD ["--config", "/app/config.json"]

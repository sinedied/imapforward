# Build stage
FROM golang:1.22-alpine AS build
ARG APP_VERSION=0.0.0-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY src/ ./src/
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=${APP_VERSION}" -o /app/imapforward ./src/

# Production stage
FROM alpine:3
RUN apk add --no-cache ca-certificates && \
    addgroup -g 1001 -S appgroup && \
    adduser -S appuser -u 1001 -G appgroup
WORKDIR /app
COPY --from=build /app/imapforward .
USER appuser

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["./imapforward"]
CMD ["-config", "/app/config.json"]

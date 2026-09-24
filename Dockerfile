# syntax=docker/dockerfile:1

# ---- 前端 ----
FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ---- 後端 ----
FROM golang:1.26.6-alpine AS go
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server . \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# ---- 執行 ----
FROM alpine:3.22
RUN apk add --no-cache ca-certificates \
 && adduser -D -H -u 10001 app
WORKDIR /app
COPY --from=go /out/server /out/migrate ./
COPY --from=web /src/web/dist ./web/dist
# 程式以相對路徑寫 logs/app.log、讀 web/dist，工作目錄需可寫
RUN mkdir logs && chown app:app logs
USER app

# 設定一律走環境變數（docker run --env-file .env），映像內不放 .env
ENV APP_ENV=production SERVER_PORT=8080
EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -qO- "http://127.0.0.1:${SERVER_PORT}/healthz" >/dev/null || exit 1
CMD ["./server"]

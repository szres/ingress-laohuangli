# Stage 1: 构建前端
FROM node:18-alpine AS frontend
WORKDIR /app/website
COPY website/package*.json ./
RUN npm ci
COPY website/ .
RUN npm run build

# Stage 2: 构建 Go 后端
FROM golang:alpine AS backend
WORKDIR /app
COPY tgbot/go.mod tgbot/go.sum ./
RUN go mod download
COPY tgbot/ .
COPY --from=frontend /app/website/build ./dist
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /app/laohuangli

# Stage 3: 最终镜像
FROM scratch
COPY --from=backend /app/laohuangli /app/laohuangli
COPY --from=backend /app/dist /app/dist
COPY --from=backend /etc/ssl/certs /etc/ssl/certs
VOLUME /db
EXPOSE 80
ENTRYPOINT ["/app/laohuangli"]

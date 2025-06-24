
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache build-base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /main .

FROM alpine:latest

# 从构建器阶段复制编译好的二进制文件
COPY --from=builder /main /main

# 暴露端口
EXPOSE 8080

# 设置入口点
ENTRYPOINT ["/main"]
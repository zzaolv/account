
FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download


COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /main .

FROM scratch

COPY --from=builder /main /main

COPY simple_ledger.db /simple_ledger.db

# 暴露端口
EXPOSE 8080

ENTRYPOINT ["/main"]
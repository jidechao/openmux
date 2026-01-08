# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 安装依赖
RUN apk add --no-cache git

# 复制 go mod 文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o openmux ./cmd/server

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# 从 builder 复制二进制文件
COPY --from=builder /app/openmux .

# 暴露端口
EXPOSE 8080

# 运行
CMD ["./openmux", "-config", "/app/config.yaml"]

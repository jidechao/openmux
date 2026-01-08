.PHONY: build run test clean docker-build docker-run

# 构建
build:
	go build -o bin/openmux ./cmd/server

# 运行
run:
	go run ./cmd/server -config config.yaml

# 测试
test:
	go test -v ./...

# 清理
clean:
	rm -rf bin/

# Docker 构建
docker-build:
	docker build -t openmux:latest .

# Docker 运行
docker-run:
	docker-compose up -d

# Docker 停止
docker-stop:
	docker-compose down

# 格式化代码
fmt:
	go fmt ./...

# 代码检查
lint:
	golangci-lint run

# 下载依赖
deps:
	go mod download
	go mod tidy

# 生成 go.sum
tidy:
	go mod tidy

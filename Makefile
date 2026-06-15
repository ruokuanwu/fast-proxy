.PHONY: help build install uninstall run test fmt tidy clean

APP_NAME := fast-proxy
CMD_DIR := ./cmd/fast-proxy
BIN_DIR := bin
BIN := $(BIN_DIR)/$(APP_NAME)
PREFIX ?= /usr/local

help: ## 显示可用命令
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "%-12s %s\n", $$1, $$2}'

build: ## 构建二进制文件到 bin/fast-proxy
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN) $(CMD_DIR)

install: build ## 安装到 $(PREFIX)/bin，需要权限时请使用 sudo
	sudo install -m 0755 $(BIN) $(PREFIX)/bin/$(APP_NAME)
	sudo ln -sf $(PREFIX)/bin/$(APP_NAME) $(PREFIX)/bin/fp

uninstall: ## 从 $(PREFIX)/bin 卸载，需要权限时请使用 sudo
	sudo rm -f $(PREFIX)/bin/$(APP_NAME)
	sudo rm -f $(PREFIX)/bin/fp

run: ## 运行 CLI，可通过 ARGS="list" 传参
	go run $(CMD_DIR) $(ARGS)

test: ## 运行测试
	go test ./...

fmt: ## 格式化 Go 代码
	go fmt ./...

tidy: ## 整理 Go 依赖
	go mod tidy

clean: ## 清理构建产物
	rm -rf $(BIN_DIR)

.PHONY: help build install uninstall run test fmt tidy clean

APP_NAME := fast-proxy
CMD_DIR := ./cmd/fast-proxy
BIN_DIR := bin
BIN := $(BIN_DIR)/$(APP_NAME)
PREFIX ?= /usr/local

help: ## Show available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "%-12s %s\n", $$1, $$2}'

build: ## Build the binary into bin/fast-proxy
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN) $(CMD_DIR)

install: build ## Install into $(PREFIX)/bin; use sudo if permissions are required
	sudo install -m 0755 $(BIN) $(PREFIX)/bin/$(APP_NAME)
	sudo ln -sf $(PREFIX)/bin/$(APP_NAME) $(PREFIX)/bin/fp

uninstall: ## Uninstall from $(PREFIX)/bin; use sudo if permissions are required
	sudo rm -f $(PREFIX)/bin/$(APP_NAME)
	sudo rm -f $(PREFIX)/bin/fp

run: ## Run the CLI; pass arguments with ARGS="list"
	go run $(CMD_DIR) $(ARGS)

test: ## Run tests
	go test ./...

fmt: ## Format Go code
	go fmt ./...

tidy: ## Tidy Go dependencies
	go mod tidy

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)

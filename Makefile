.PHONY: build run test clean deps lint fmt help

# 変数定義
BINARY_NAME=gem-server
CMD_PATH=cmd/main.go
BUILD_DIR=bin
COVERAGE_DIR=coverage

# デフォルトターゲット
.DEFAULT_GOAL := help

## build: アプリケーションをビルド
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## run: アプリケーションを実行
run:
	@echo "Running $(BINARY_NAME)..."
	@go run $(CMD_PATH)

## test: テストを実行
test:
	@echo "Running tests..."
	@go test -v ./...

## test-coverage: カバレッジ付きテストを実行
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	@go test -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	@go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report generated: $(COVERAGE_DIR)/coverage.html"

## deps: 依存パッケージをインストール
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

## lint: コードをリント
lint:
	@echo "Running linter..."
	@go vet ./...
	@golangci-lint run ./... || echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"

## fmt: コードをフォーマット
fmt:
	@echo "Formatting code..."
	@go fmt ./...

## clean: ビルド成果物を削除
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(COVERAGE_DIR)
	@go clean

## help: このヘルプメッセージを表示
help:
	@echo "Available targets:"
	@grep -E '^##' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'

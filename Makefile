
# Build settings
BINARY_NAME=go-create
MAIN_PATH=./cmd/create
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}"

# Environment settings for Mac Intel
GOOS=darwin
GOARCH=amd64

# Tools and commands
GO=go
GOFMT=$(GO) fmt
GOVET=$(GO) vet
GOLINT=golangci-lint

.PHONY: all build clean test fmt lint vet install help

all: clean fmt lint vet test build

build:
	@echo "Building ${BINARY_NAME} for macOS (Intel)..."
	@mkdir -p bin
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build $(LDFLAGS) -o bin/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Binary created at bin/$(BINARY_NAME)"

clean:
	@echo "Cleaning up..."
	@rm -rf bin/
	@rm -f $(BINARY_NAME)
	@rm -rf coverage.out
	@rm -rf vendor/
	@go clean -cache
	@echo "Clean complete"

test:
	@echo "Running tests..."
	$(GO) test -race -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

lint:
	@if command -v $(GOLINT) &> /dev/null; then \
		echo "Running linter..."; \
		$(GOLINT) run; \
	else \
		echo "golangci-lint not installed. Skipping lint."; \
		echo "To install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

vet:
	@echo "Running go vet..."
	$(GOVET) ./...

install:
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "Dependencies installed"

run: build
	@echo "Running ${BINARY_NAME}..."
	./bin/$(BINARY_NAME)

# Cross-compilation targets
build-linux:
	@echo "Building ${BINARY_NAME} for Linux..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	@echo "Binary created at bin/$(BINARY_NAME)-linux-amd64"

build-windows:
	@echo "Building ${BINARY_NAME} for Windows..."
	@mkdir -p bin
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o bin/$(BINARY_NAME).exe $(MAIN_PATH)
	@echo "Binary created at bin/$(BINARY_NAME).exe"

build-all: build build-linux build-windows

help:
	@echo "Available commands:"
	@echo "  make build       - Build for macOS (Intel)"
	@echo "  make clean       - Remove build artifacts"
	@echo "  make test        - Run tests"
	@echo "  make fmt         - Format code"
	@echo "  make lint        - Run linter"
	@echo "  make vet         - Run go vet"
	@echo "  make install     - Install dependencies"
	@echo "  make run         - Build and run the application"
	@echo "  make build-linux - Build for Linux"
	@echo "  make build-windows - Build for Windows"
	@echo "  make build-all   - Build for all platforms"
	@echo "  make all         - Run clean, fmt, lint, vet, test, and build"
	@echo "  make help        - Show this help"

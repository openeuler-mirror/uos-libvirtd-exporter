.PHONY: build clean test fmt lint run help

# Binary name
BINARY_NAME=uos-libvirtd-exporter

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt

# Build flags
LDFLAGS=-ldflags "-s -w"

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) -v

clean: ## Clean build artifacts
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

test: ## Run tests
	$(GOTEST) -v ./...

fmt: ## Format code
	$(GOFMT) -s -w .

lint: ## Run linter (requires golangci-lint)
	golangci-lint run

deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) tidy

run: ## Run the exporter (requires libvirt)
	./$(BINARY_NAME) -libvirt.uri=qemu:///system

install: build ## Install the binary to /usr/local/bin
	sudo cp $(BINARY_NAME) /usr/local/bin/

uninstall: ## Remove the binary from /usr/local/bin
	sudo rm -f /usr/local/bin/$(BINARY_NAME)

# Development helpers
dev-setup: ## Setup development environment (install dependencies)
	@echo "Installing development dependencies..."
	$(GOGET) -u golang.org/x/tools/cmd/goimports
	@echo "Development setup complete."

# Cross compilation
build-linux: ## Build for Linux
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 -v

build-arm64: ## Build for ARM64
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 -v

# Docker support (optional)
docker-build: ## Build Docker image
	docker build -t $(BINARY_NAME):latest .

docker-run: ## Run in Docker container
	docker run -p 9177:9177 --privileged $(BINARY_NAME):latest
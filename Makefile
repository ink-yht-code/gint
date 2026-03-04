.PHONY: all build test lint clean gen install

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary names
GINT_GEN_BIN=gint-gen
REGISTRY_BIN=registry

# Directories
GINT_DIR=./gint
GINTX_DIR=./gintx
GINT_GEN_DIR=./gint-gen
REGISTRY_DIR=./registry
EXAMPLE_DIR=./example/user

# Build all modules
all: build

# Build all modules
build:
	@echo "Building gint..."
	cd $(GINT_DIR) && $(GOBUILD) .
	@echo "Building gintx..."
	cd $(GINTX_DIR) && $(GOBUILD) ./...
	@echo "Building gint-gen..."
	cd $(GINT_GEN_DIR) && $(GOBUILD) -o bin/$(GINT_GEN_BIN) .
	@echo "Building registry..."
	cd $(REGISTRY_DIR) && $(GOBUILD) -o bin/$(REGISTRY_BIN) ./cmd/...
	@echo "Build complete!"

# Run tests
test:
	@echo "Running tests..."
	cd $(GINT_DIR) && $(GOTEST) -v ./...
	cd $(GINTX_DIR) && $(GOTEST) -v ./...
	cd $(GINT_GEN_DIR) && $(GOTEST) -v ./...
	cd $(REGISTRY_DIR) && $(GOTEST) -v ./...

# Run linter
lint:
	@echo "Running linter..."
	$(GOCMD) vet ./...
	@echo "Running architecture lint..."
	cd $(GINT_GEN_DIR) && $(GOCMD) run . lint $(EXAMPLE_DIR)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(GINT_GEN_DIR)/bin
	rm -rf $(REGISTRY_DIR)/bin
	rm -f $(EXAMPLE_DIR)/main.exe
	$(GOCLEAN)

# Tidy dependencies
tidy:
	cd $(GINT_DIR) && $(GOMOD) tidy
	cd $(GINTX_DIR) && $(GOMOD) tidy
	cd $(GINT_GEN_DIR) && $(GOMOD) tidy
	cd $(REGISTRY_DIR) && $(GOMOD) tidy
	cd $(EXAMPLE_DIR) && $(GOMOD) tidy

# Install gint-gen to GOPATH/bin
install:
	cd $(GINT_GEN_DIR) && $(GOCMD) install .

# Generate example service
gen-example:
	cd $(GINT_GEN_DIR) && $(GOCMD) run . new service user --transport http

# Run registry service
run-registry:
	cd $(REGISTRY_DIR) && $(GOCMD) run ./cmd/...

# Run example service
run-example:
	cd $(EXAMPLE_DIR) && $(GOCMD) run ./cmd/main.go

# Build example service
build-example:
	cd $(EXAMPLE_DIR) && $(GOBUILD) -o bin/user ./cmd/main.go

# Help
help:
	@echo "Available targets:"
	@echo "  all          - Build all modules"
	@echo "  build        - Build all modules"
	@echo "  test         - Run all tests"
	@echo "  lint         - Run linter"
	@echo "  clean        - Clean build artifacts"
	@echo "  tidy         - Tidy dependencies"
	@echo "  install      - Install gint-gen to GOPATH/bin"
	@echo "  gen-example  - Generate example service"
	@echo "  run-registry - Run registry service"
	@echo "  run-example  - Run example service"
	@echo "  build-example- Build example service"

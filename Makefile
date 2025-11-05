# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet
BINARY_NAME=spot-instance-advisor
# Detect host go env for naming
OS:=$(shell $(GOCMD) env GOOS)
ARCH:=$(shell $(GOCMD) env GOARCH)
EXT:=$(if $(filter $(OS),windows),.exe,)

.PHONY: all build build-release clean test deps fmt vet tidy vendor help

all: deps test build

# Build the binary (with OS-ARCH suffix)
build:
	@mkdir -p dist
	$(GOBUILD) -o dist/$(BINARY_NAME)-$(OS)-$(ARCH)$(EXT) -v .

# Build for release (optimized, with OS-ARCH suffix)
build-release:
	@echo "Building optimized release binary..."
	@mkdir -p dist
	CGO_ENABLED=0 $(GOBUILD) -ldflags="-s -w" -o dist/$(BINARY_NAME)-$(OS)-$(ARCH)$(EXT) -v .
	@echo "Release build completed"
	@ls -la dist/

# Build for all platforms (cross-compilation)
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p dist
	# Linux
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) -ldflags="-s -w" -o dist/$(BINARY_NAME)-linux-amd64 -v .
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 $(GOBUILD) -ldflags="-s -w" -o dist/$(BINARY_NAME)-linux-arm64 -v .
	# macOS
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) -ldflags="-s -w" -o dist/$(BINARY_NAME)-darwin-amd64 -v .
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 $(GOBUILD) -ldflags="-s -w" -o dist/$(BINARY_NAME)-darwin-arm64 -v .
	# Windows
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) -ldflags="-s -w" -o dist/$(BINARY_NAME)-windows-amd64.exe -v .
	@echo "All builds completed"
	@ls -la dist/



# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -rf dist/

# Run tests
test: fmt vet
	$(GOTEST) ./... -coverprofile cover.out

# Run tests with coverage
test-coverage: test
	$(GOCMD) tool cover -html=cover.out

# Run go fmt against code
fmt:
	$(GOFMT) ./...

# Run go vet against code
vet:
	$(GOVET) ./...

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Update dependencies
deps-update:
	$(GOGET) -u ./...
	$(GOMOD) tidy

# Create vendor directory (optional)
vendor:
	$(GOMOD) vendor

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  build-release  - Build optimized release binary"
	@echo "  build-all     - Build for all platforms (cross-compilation)"
	@echo "  clean         - Clean build artifacts"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  fmt           - Format code"
	@echo "  vet           - Run go vet"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  deps-update   - Update dependencies"
	@echo "  vendor        - Create vendor directory"
	@echo "  help          - Show this help"


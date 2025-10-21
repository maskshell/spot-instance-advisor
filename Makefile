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

.PHONY: all build clean test deps fmt vet tidy vendor help

all: deps test build

# Build the binary
build:
	$(GOBUILD) -o $(BINARY_NAME) -v .



# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

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
	@echo "  clean         - Clean build artifacts"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  fmt           - Format code"
	@echo "  vet           - Run go vet"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  deps-update   - Update dependencies"
	@echo "  vendor        - Create vendor directory"
	@echo "  help          - Show this help"


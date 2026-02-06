.PHONY: build test test-verbose test-race coverage coverage-html lint clean deps check install calibrate-providers help

# Binary name
BINARY=nightshift
PKG=./cmd/nightshift

# Build the binary
build:
	go build -o $(BINARY) $(PKG)

# Install the binary to your Go bin directory
install:
	go install $(PKG)
	@echo "Installed $(BINARY) to $$(if [ -n "$$(go env GOBIN)" ]; then go env GOBIN; else echo "$$(go env GOPATH)/bin"; fi)"

# Run provider calibration comparison tool
calibrate-providers:
	go run ./cmd/provider-calibration --repo "$$(pwd)" --codex-originator codex_cli_rs --min-user-turns 2

# Run all tests
test:
	go test ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run tests with race detection
test-race:
	go test -race ./...

# Run tests with coverage report
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@echo ""
	@echo "To view HTML coverage report, run: go tool cover -html=coverage.out"

# Generate HTML coverage report
coverage-html: coverage
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run golangci-lint (if installed)
lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run

# Clean build artifacts
clean:
	rm -f $(BINARY)
	rm -f coverage.out
	rm -f coverage.html

# Install development dependencies
deps:
	go mod download
	go mod tidy

# Run all checks (test + lint)
check: test lint

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  test          - Run all tests"
	@echo "  test-verbose  - Run tests with verbose output"
	@echo "  test-race     - Run tests with race detection"
	@echo "  coverage      - Run tests with coverage report"
	@echo "  coverage-html - Generate HTML coverage report"
	@echo "  lint          - Run golangci-lint"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  check         - Run tests and lint"
	@echo "  install       - Build and install to Go bin directory"
	@echo "  calibrate-providers - Compare local Claude/Codex session usage for calibration"
	@echo "  help          - Show this help"

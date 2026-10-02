.PHONY: build fmt vet test test-verbose test-race coverage coverage-html lint lint-install clean deps check install calibrate-providers install-hooks help

# Binary name
BINARY=nightshift
PKG=./cmd/nightshift
GO=go
GOLANGCI_LINT=golangci-lint
GOLANGCI_LINT_CONFIG=.golangci.yml
GOLANGCI_LINT_VERSION=v1.64.8
GO_ROOT_BIN=$(shell $(GO) env GOROOT)/bin
GOFMT=$(GO_ROOT_BIN)/gofmt
GO_BIN_OVERRIDE=$(shell $(GO) env GOBIN)
GO_BIN_DIR=$(if $(GO_BIN_OVERRIDE),$(GO_BIN_OVERRIDE),$(shell $(GO) env GOPATH)/bin)
GOLANGCI_LINT_BIN=$(GO_BIN_DIR)/$(GOLANGCI_LINT)

# Build the binary
build:
	$(GO) build -o $(BINARY) $(PKG)

# Install the binary to your Go bin directory
install:
	$(GO) install $(PKG)
	@echo "Installed $(BINARY) to $$(if [ -n "$$($(GO) env GOBIN)" ]; then $(GO) env GOBIN; else echo "$$($(GO) env GOPATH)/bin"; fi)"

# Run provider calibration comparison tool
calibrate-providers:
	$(GO) run ./cmd/provider-calibration --repo "$$(pwd)" --codex-originator codex_cli_rs --min-user-turns 2

# Check formatting without rewriting files
fmt:
	@GO_FILES="$$(git ls-files -- '*.go')"; \
	if [ -z "$$GO_FILES" ]; then \
		echo "gofmt: no Go files"; \
	else \
		UNFORMATTED="$$( $(GOFMT) -l $$GO_FILES )" || exit 1; \
		if [ -z "$$UNFORMATTED" ]; then \
			echo "gofmt: ok"; \
		else \
			echo "gofmt: run 'gofmt -w' on:"; \
			echo "$$UNFORMATTED"; \
			exit 1; \
		fi; \
	fi

# Run go vet across the module
vet:
	$(GO) vet ./...

# Run all tests
test:
	$(GO) test ./...

# Run tests with verbose output
test-verbose:
	$(GO) test -v ./...

# Run tests with race detection
test-race:
	$(GO) test -race ./...

# Run tests with coverage report
coverage:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out
	@echo ""
	@echo "To view HTML coverage report, run: go tool cover -html=coverage.out"

# Generate HTML coverage report
coverage-html: coverage
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run golangci-lint with the checked-in config
lint:
	@if command -v $(GOLANGCI_LINT) > /dev/null; then \
		LINT_BIN="$$(command -v $(GOLANGCI_LINT))"; \
	elif [ -x "$(GOLANGCI_LINT_BIN)" ]; then \
		LINT_BIN="$(GOLANGCI_LINT_BIN)"; \
	else \
		echo "golangci-lint not installed. Run: make lint-install"; \
		exit 1; \
	fi; \
	PATH="$(GO_ROOT_BIN):$$PATH" "$$LINT_BIN" run --config $(GOLANGCI_LINT_CONFIG) ./...

# Install the repo's pinned golangci-lint version
lint-install:
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

# Clean build artifacts
clean:
	rm -f $(BINARY)
	rm -f coverage.out
	rm -f coverage.html

# Install development dependencies
deps:
	$(GO) mod download
	$(GO) mod tidy

# Run the full local verification suite
check: fmt vet test lint

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  fmt           - Check gofmt output"
	@echo "  vet           - Run go vet"
	@echo "  test          - Run all tests"
	@echo "  test-verbose  - Run tests with verbose output"
	@echo "  test-race     - Run tests with race detection"
	@echo "  coverage      - Run tests with coverage report"
	@echo "  coverage-html - Generate HTML coverage report"
	@echo "  lint          - Run golangci-lint"
	@echo "  lint-install  - Install pinned golangci-lint ($(GOLANGCI_LINT_VERSION))"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  check         - Run fmt, vet, tests, and lint"
	@echo "  install       - Build and install to Go bin directory"
	@echo "  calibrate-providers - Compare local Claude/Codex session usage for calibration"
	@echo "  install-hooks  - Install git pre-commit hook"
	@echo "  help          - Show this help"

# Install git pre-commit hook
install-hooks:
	@ln -sf ../../scripts/pre-commit.sh .git/hooks/pre-commit
	@echo "✓ pre-commit hook installed (.git/hooks/pre-commit → scripts/pre-commit.sh)"

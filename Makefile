GOBIN := $(shell go env GOPATH)/bin

.PHONY: all test build clean lint fmt coverage coverage-check check tidy deps install

# Default target
all: fmt lint test build

# Run tests
test:
	go test -v -race ./...

# Run tests with coverage
coverage:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"
	@go tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'

# Check if coverage meets minimum threshold (80%)
coverage-check:
	@go test -race -coverprofile=coverage.out -covermode=atomic ./... > /dev/null
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ $$(echo "$$COVERAGE < 80" | bc -l) -eq 1 ]; then \
		echo "Coverage $$COVERAGE% is below minimum 80%"; \
		exit 1; \
	else \
		echo "Coverage $$COVERAGE% meets minimum 80%"; \
	fi

# Build the application
build:
	go build -v -o claude-config-merge ./cmd/claude-config-merge

# Format code
fmt:
	gofmt -w -s .
	$(GOBIN)/goimports -w .

# Run linters
lint:
	golangci-lint run ./...

# Clean build artifacts
clean:
	go clean
	rm -f coverage.out coverage.html

# Tidy dependencies
tidy:
	go mod tidy
	go mod verify

# Run all checks (fmt, lint, test with coverage check)
check: fmt lint coverage-check

# Install binary to ~/.local/bin
install: build
	mkdir -p $(HOME)/.local/bin
	cp claude-config-merge $(HOME)/.local/bin/claude-config-merge

# Install development dependencies
deps:
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

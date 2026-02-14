# Simple Makefile for a Go project

# Build the application
all: build test

build:
	@echo "Building Backend..."
	@go build -o main cmd/api/main.go
	@echo "Installing Frontend Dependencies..."
	@npm install --prefer-offline --no-fund --prefix ./frontend

# Run the application (Expects 'make build' to be run first)
start:
	@echo "Starting Backend..."
	@./main & \
	echo "Starting Frontend..." && \
	npm run dev --prefix ./frontend

# Run the application (Production)
run:
	@./main

# Test the application
test:
	@echo "Testing..."
	@go test ./... -v

# Clean the binary
clean:
	@echo "Cleaning..."
	@rm -f main

# Live Reload
watch:
	@if command -v air > /dev/null; then \
            air; \
            echo "Watching...";\
        else \
            read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
            if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
                go install github.com/air-verse/air@latest; \
                air; \
                echo "Watching...";\
            else \
                echo "You chose not to install air. Exiting..."; \
                exit 1; \
            fi; \
        fi

.PHONY: all build run test clean watch

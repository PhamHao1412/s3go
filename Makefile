.PHONY: all build run test clean

# Default target
all: build

# Build target: Compiles the Go application
build:
	@echo "==> Building S3Go binary..."
	go build -o s3go main.go
	@echo "==> Build successful! Binary created: ./s3go"

# Run target: Builds and then runs the application
run: build
	@echo "==> Starting S3Go server..."
	./s3go

# Test target: Runs unit tests
test:
	@echo "==> Running unit tests..."
	go test -v ./internal/crypto/...

# Clean target: Cleans build artifacts and temp files
clean:
	@echo "==> Cleaning build artifacts..."
	rm -f s3go
	rm -rf tmp/
	@echo "==> Clean complete."

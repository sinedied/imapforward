# Build the binary
build:
    go build -o imapforward ./src/

# Run directly (no build needed)
run *args:
    go run ./src/ {{args}}

# Run all tests
test:
    go test -count=1 ./src/

# Run tests with verbose output
test-v:
    go test -v -count=1 ./src/

# Run static analysis
vet:
    go vet ./src/

# Run linter
lint:
    golangci-lint run ./src/

# Format code
fmt:
    gofmt -w ./src/

# Run build + vet + test + lint (full check)
check: build vet test lint
    @echo "All checks passed"

# Build Docker image
docker-build tag="imapforward":
    docker build -t {{tag}} .

# Run with Docker
docker-run:
    docker run --rm -v ./config.json:/app/config.json imapforward

# Clean build artifacts
clean:
    rm -f imapforward

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

# Prepare release: update version in source and cross-compile binaries
release-prepare version:
    sed -i'' -e 's/var version = ".*"/var version = "{{version}}"/' src/main.go
    just build-all {{version}}

# Cross-compile binaries for all platforms
build-all version="dev":
    mkdir -p dist
    for pair in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64; do \
        os=${pair%/*}; arch=${pair#*/}; ext=""; \
        [ "$os" = "windows" ] && ext=".exe"; \
        CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -ldflags="-s -w -X main.version={{version}}" -o "dist/imapforward-${os}-${arch}${ext}" ./src/; \
    done

# Clean build artifacts
clean:
    rm -f imapforward
    rm -rf dist

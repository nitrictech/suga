
test:
	@echo "Running tests..."
	@go test ./terraform/...

test-verbose:
	@echo "Running tests with verbose output..."
	@go test -v ./terraform/...

test-short:
	@echo "Running tests with short output..."
	@go test -short ./terraform/...

test-pretty:
	@go tool gotest.tools/gotestsum --format=pkgname ./terraform/...
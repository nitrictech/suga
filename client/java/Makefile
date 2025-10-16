.PHONY: clean compile test package install deploy help

# Default target
all: package

# Clean the project
clean:
	mvn clean

# Compile the project
compile:
	mvn compile

# Generate protobuf sources
gen-proto:
	mvn protobuf:compile protobuf:compile-custom

# Run tests
test:
	mvn test

# Package the project
package:
	mvn package

# Install to local repository
install:
	mvn install

# Deploy to remote repository (requires proper configuration)
deploy:
	mvn deploy

# Format code (if using spotless or similar)
format:
	mvn spotless:apply

# Check code formatting
check-format:
	mvn spotless:check

# Run all checks
verify:
	mvn verify

# Show help
help:
	@echo "Available targets:"
	@echo "  clean        - Clean the project"
	@echo "  compile      - Compile the project"
	@echo "  gen-proto    - Generate protobuf sources"
	@echo "  test         - Run tests"
	@echo "  package      - Package the project"
	@echo "  install      - Install to local repository"
	@echo "  deploy       - Deploy to remote repository"
	@echo "  verify       - Run all checks"
	@echo "  help         - Show this help message"
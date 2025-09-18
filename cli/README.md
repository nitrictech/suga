# Suga CLI Development

This directory contains the Suga CLI implementation. This README is for developers working on the CLI codebase.

> **For CLI usage documentation**, see [docs.addsuga.com/cli](https://docs.addsuga.com/cli)

## Table of Contents

- [Development Setup](#development-setup)
- [Architecture](#architecture)
- [Building](#building)
- [Testing](#testing)
- [Development Workflow](#development-workflow)
- [Adding New Commands](#adding-new-commands)
- [Debugging](#debugging)
- [Contributing](#contributing)

## Development Setup

### Prerequisites

- **Go 1.24+** (required)
- **Make** (for build automation)
- **Git** (for version info injection)

### Getting Started

1. **Clone the repository**
   ```bash
   git clone https://github.com/nitrictech/suga.git
   cd suga/cli
   ```

2. **Install dependencies**
   ```bash
   make install
   ```

3. **Build the CLI**
   ```bash
   make build
   ```

4. **Run the CLI**
   ```bash
   ./bin/suga version
   ```

## Architecture

The CLI follows a modular architecture with clear separation of concerns:

```
cli/
├── main.go              # Entry point and dependency injection setup
├── cmd/                 # Command definitions (Cobra commands)
│   ├── root.go         # Root command and command registration
│   ├── auth.go         # Authentication commands (login, logout, access-token)
│   ├── nitric.go       # Core commands (new, init, build, dev, etc.)
│   ├── config.go       # Configuration commands
│   └── team.go         # Team management commands
├── internal/           # Internal packages
│   ├── api/           # Suga API client and models
│   ├── browser/       # Browser automation for auth flows
│   ├── build/         # Build system and Terraform generation
│   ├── config/        # Configuration management
│   ├── devserver/     # Local development server
│   ├── netx/          # Network utilities
│   ├── platforms/     # Platform/target management
│   ├── plugins/       # Plugin system
│   ├── simulation/    # Local resource simulation
│   ├── style/         # CLI styling and colors
│   ├── utils/         # Utility functions
│   ├── version/       # Version information
│   └── workos/        # WorkOS authentication integration
└── pkg/               # Public packages (if any)
```

### Key Design Patterns

1. **Dependency Injection**: Uses `samber/do/v2` for dependency management
2. **Command Pattern**: Cobra commands in `cmd/` package
3. **Repository Pattern**: API clients and data access in `internal/api/`
4. **Factory Pattern**: Service construction in `main.go`

### Core Dependencies

- **CLI Framework**: `spf13/cobra` for command structure
- **Configuration**: `spf13/viper` for config management
- **DI Container**: `samber/do/v2` for dependency injection
- **UI Components**: `charmbracelet/huh` and `charmbracelet/lipgloss` for TUI
- **Auth**: `zalando/go-keyring` for secure token storage

## Building

### Make Targets

```bash
make build         # Build the CLI binary
make install       # Install/update dependencies
make test          # Run tests
make test-verbose  # Run tests with verbose output
make test-short    # Run short tests only
make fmt           # Format and fix code style
make lint          # Run linters
```

### Build Process

The build process injects version information via LDFLAGS:

```bash
# Version info is injected at build time
BUILD_VERSION=$(git describe --tags --abbrev=7 --dirty)
SOURCE_GIT_COMMIT=$(git rev-parse --short --dirty HEAD)
BUILD_TIME=$(date +%Y-%m-%dT%H:%M:%S%z)
```

Binary output: `bin/suga` (or `bin/suga.exe` on Windows)

### Cross-Platform Builds

The Makefile automatically handles Windows executable extensions:

```makefile
ifeq ($(OS), Windows_NT)
    EXECUTABLE_EXT = .exe
else
    EXECUTABLE_EXT =
endif
```

## Testing

### Running Tests

```bash
# Basic test run
make test

# Verbose output
make test-verbose

# Short tests (skip integration tests)
make test-short

# Pretty output with gotestsum
make test-pretty
```

### Test Structure

- Tests are located alongside source files (`*_test.go`)
- Integration tests should use build tag `// +build integration`
- Mock external dependencies in tests

### Coverage

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Development Workflow

### 1. Adding Features

1. Create feature branch: `git checkout -b feature/my-feature`
2. Implement changes in appropriate packages
3. Add tests for new functionality
4. Update documentation if needed
5. Run `make fmt` and `make lint`
6. Test locally: `make build && ./bin/suga [command]`
7. Submit pull request

### 2. Local Development Loop

```bash
# 1. Make changes to source code
# 2. Build and test
make build && ./bin/suga version

# 3. Test specific functionality
./bin/suga new test-project --force

# 4. Run tests
make test

# 5. Format and lint
make fmt && make lint
```

### 3. Debugging Local Changes

```bash
# Build with debug info
go build -ldflags "$(LDFLAGS)" -gcflags="all=-N -l" -o bin/suga-debug main.go

# Use with debugger (delve)
dlv exec ./bin/suga-debug -- [args]
```

## Adding New Commands

### 1. Create Command Function

Add to appropriate file in `cmd/` (or create new file):

```go
// cmd/mycommand.go
func NewMyCommand(injector do.Injector) *cobra.Command {
    return &cobra.Command{
        Use:   "mycommand",
        Short: "Description of my command",
        Long:  "Longer description...",
        Run: func(cmd *cobra.Command, args []string) {
            // Implementation
        },
    }
}
```

### 2. Register Command

Add to `cmd/root.go` in `NewRootCmd()`:

```go
rootCmd.AddCommand(NewMyCommand(injector))
```

### 3. Add Business Logic

Create service in `internal/` if needed:

```go
// internal/myservice/myservice.go
type MyService struct {
    // dependencies
}

func NewMyService(deps...) *MyService {
    return &MyService{...}
}

func (s *MyService) DoSomething() error {
    // implementation
}
```

### 4. Wire Dependencies

Add to `main.go` if new service is needed:

```go
do.Provide(injector, myservice.NewMyService)
```

## Recent Features

### Generate Command Configuration Support

The generate command now supports configuration-based client generation via the `suga.yaml` file. This feature was added to simplify the developer experience by allowing users to define their client generation preferences once in the config file rather than typing long command lines repeatedly.

**Implementation Details:**

1. **Schema Changes** (`pkg/schema/schema.go`):
   - Added `GenerateConfig` struct with `Language`, `OutputPath`, and `PackageName` fields
   - Added `Generate []GenerateConfig` field to `Application` struct
   - JSON schema validation for supported languages: `go`, `python`, `ts`, `js`

2. **Command Logic** (`cmd/nitric.go`):
   - Modified `NewGenerateCmd()` to check if any language flags are provided
   - If no flags provided, calls `app.GenerateFromConfig()` instead of the traditional flag-based method
   - Command-line flags take precedence over configuration for backward compatibility

3. **Business Logic** (`pkg/app/nitric.go`):
   - Added `GenerateFromConfig()` method that reads config from `suga.yaml`
   - Validates that generate configuration exists and contains valid entries
   - Processes each generate configuration entry sequentially
   - Provides detailed error messages for missing config or validation failures

4. **Validation** (`pkg/schema/validate.go`):
   - Added `checkGenerateConfigurations()` validation method
   - Validates supported languages (`go`, `python`, `ts`, `js`)
   - Prevents duplicate language configurations
   - Validates Go package names (lowercase, alphanumeric only)
   - Ensures required `output_path` is provided

**Usage Example:**
```yaml
# suga.yaml
generate:
  - language: python
    output_path: ./suga_gen
  - language: go
    output_path: ./pkg
    package_name: suga
  - language: ts
    output_path: ./src
```

**Testing:**
- ✅ Config-based generation when no flags provided
- ✅ Validation of invalid languages, duplicate configs, and malformed package names
- ✅ Helpful error messages when no config exists and no flags provided
- ✅ Command-line flags still work and take precedence over config

## Debugging

### 1. CLI Debug Output

Enable verbose logging:

```bash
# Set debug environment
export SUGA_DEBUG=true

# Or use verbose flags where available
./bin/suga [command] --verbose
```

### 2. Development Server Debugging

For `suga dev` debugging:

```bash
# Check what's running
lsof -i :50051  # Default gRPC port
lsof -i :3000   # Default HTTP port

# View logs
tail -f ~/.suga/logs/dev.log
```

### 3. API Debugging

For API client debugging:

```go
// In internal/api/api.go, add logging
import "log"

func (c *Client) makeRequest(...) {
    log.Printf("API Request: %s %s", method, url)
    // ... existing code
}
```

### 4. Build Debugging

For build system debugging:

```bash
# Check generated files
ls -la .suga/stacks/

# Validate Terraform
cd .suga/stacks/[stack-name]
terraform validate
terraform plan
```

## Contributing

### Code Style

- Follow `gofmt` formatting (enforced by `make fmt`)
- Use meaningful variable and function names
- Add comments for exported functions and types
- Keep functions focused and small

### Commit Guidelines

- Use conventional commits: `feat:`, `fix:`, `docs:`, `refactor:`, etc.
- Reference issues: `fixes #123`
- Keep commits atomic and focused

### Pull Request Process

1. Fork the repository
2. Create feature branch from `main`
3. Make changes with tests
4. Run `make fmt && make lint && make test`
5. Update documentation if needed
6. Submit PR with clear description

### Code Review Checklist

- [ ] Code follows Go conventions
- [ ] Tests added for new functionality
- [ ] No breaking changes (or documented)
- [ ] Error handling implemented
- [ ] Documentation updated
- [ ] CLI help text accurate

---

**Related Documentation:**
- [Main Project Contributing Guidelines](../CONTRIBUTING.md)
- [Development Guidelines](../DEVELOPERS.md)
- [CLI User Documentation](https://docs.addsuga.com/cli)

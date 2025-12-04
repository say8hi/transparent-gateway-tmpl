# Development Guide

This guide covers the development workflow, tools, and best practices for contributing to the API Gateway project.

## Prerequisites

- Go 1.23 or higher
- Git
- Make
- Docker and Docker Compose (for integration tests)

## Development Tools

### Required

These tools are included in the Go toolchain:

- `go fmt` - basic code formatter
- `go vet` - static analysis tool
- `go test` - testing framework

### Recommended (Optional)

Install these tools for better development experience:

```bash
# gofumpt - stricter formatter than go fmt
go install mvdan.cc/gofumpt@latest

# goimports - auto-manage imports
go install golang.org/x/tools/cmd/goimports@latest

# golangci-lint - comprehensive linter
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## Project Setup

1. **Clone the repository**
   ```bash
   git clone <repo-url>
   cd api-gateway-tmpl
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Install git hooks**
   ```bash
   make install-hooks
   ```

4. **Copy environment configuration**
   ```bash
   cp .env.example .env
   # Edit .env with your settings
   ```

## Development Workflow

### Code Formatting

The project uses **gofumpt** (stricter than go fmt) for code formatting:

```bash
# Check if code is formatted
make fmt-check

# Format code
make fmt

# Format code and organize imports (full formatting)
make fmt-fix
```

**EditorConfig**: The project includes `.editorconfig` for consistent formatting across editors. Install the EditorConfig plugin for your editor.

### Linting

The project uses **golangci-lint** with custom configuration (`.golangci.yml`):

```bash
# Run linter
make lint

# Run linter and auto-fix issues
make lint-fix
```

### Testing

```bash
# Run unit tests
make test

# Run unit tests with verbose output
go test -v ./...

# Run integration tests (requires Docker)
docker compose -f test/docker-compose.test.yml up -d
sleep 10
go test -v -tags=integration ./test/integration/...
docker compose -f test/docker-compose.test.yml down

# Run tests with coverage
go test -coverprofile=coverage.txt -covermode=atomic ./...
go tool cover -html=coverage.txt
```

### Building

```bash
# Build binary
make build

# Run application
make run

# Clean build artifacts
make clean
```

## Code Style Guidelines

### General Principles

1. **Follow Effective Go**: https://golang.org/doc/effective_go
2. **Follow Uber Go Style Guide**: See `UBER_GO_CODESTYLE.md` (if available)
3. **Use idiomatic Go patterns**
4. **Keep functions small and focused**
5. **Write self-documenting code**

### Formatting Rules

- **Line length**: Maximum 120 characters (enforced by golines in nvim)
- **Indentation**: Tabs (Go standard)
- **Imports**: Grouped and sorted automatically
- **Comments**: Start with lowercase, be concise

### Error Handling

```go
// Good: Return errors, don't panic
func doSomething() error {
    if err := validateInput(); err != nil {
        return fmt.Errorf("failed to validate input: %w", err)
    }
    return nil
}

// Bad: Panic in library code
func doSomething() {
    if err := validateInput(); err != nil {
        panic(err) // Don't do this!
    }
}
```

### Logging

Use structured logging with Zap:

```go
// Good: Structured logging
log.Info("processing request",
    zap.String("method", r.Method),
    zap.String("path", r.URL.Path),
)

// Bad: String concatenation
log.Info("processing request: " + r.Method + " " + r.URL.Path)
```

### Testing

```go
// Good: Table-driven tests
func TestValidateToken(t *testing.T) {
    tests := []struct {
        name    string
        token   string
        wantErr bool
    }{
        {"valid token", "valid-jwt-token", false},
        {"invalid token", "invalid", true},
        {"empty token", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateToken(tt.token)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Git Workflow

### Commit Messages

Follow **Conventional Commits** format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Formatting, missing semicolons, etc.
- `refactor`: Code refactoring
- `test`: Adding tests
- `chore`: Maintenance tasks

**Example:**
```
feat(auth): add JWT token refresh endpoint

Implement token refresh functionality that allows clients to
obtain a new access token using a valid refresh token.

- add RefreshToken method to JWT manager
- implement /auth/refresh endpoint
- add integration tests for token refresh flow

Closes #123
```

### Pre-commit Hooks

Git hooks automatically run before each commit:

1. **gofumpt/go fmt** - Format checking
2. **go vet** - Static analysis
3. **go test** - Run all tests
4. **go mod tidy** - Ensure clean dependencies

If any check fails, the commit is aborted.

**Bypass hooks** (use with caution):
```bash
git commit --no-verify
```

### Branching Strategy

- `master` - stable, production-ready code
- `feature/*` - new features
- `fix/*` - bug fixes
- `refactor/*` - code refactoring

## IDE Configuration

### VS Code

Recommended settings (`.vscode/settings.json`):

```json
{
  "go.formatTool": "gofumpt",
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "workspace",
  "editor.formatOnSave": true,
  "editor.codeActionsOnSave": {
    "source.organizeImports": true
  },
  "[go]": {
    "editor.rulers": [120]
  }
}
```

### Neovim

The project formatting follows the configuration in `~/dotfiles/config/nvim/lua/configs/conform.lua`:

- **gofumpt** - stricter formatting
- **goimports-reviser** - import organization
- **golines** - line length control (max 120)

## CI/CD Pipeline

### GitLab CI Stages

1. **test** - Unit tests, formatting, linting
2. **integration-test** - Integration tests with mock services
3. **build** - Docker image build

See `.gitlab-ci.yml` for details.

## Troubleshooting

### Formatting Issues

```bash
# Check what's wrong
make fmt-check

# Fix automatically
make fmt-fix
```

### Linting Issues

```bash
# See all issues
make lint

# Auto-fix what can be fixed
make lint-fix
```

### Test Failures

```bash
# Run tests with verbose output
go test -v ./...

# Run specific test
go test -v -run TestName ./path/to/package
```

### Import Issues

```bash
# Organize imports
goimports -w .

# Or use make target
make fmt-fix
```

## Resources

- [Effective Go](https://golang.org/doc/effective_go)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
- [golangci-lint](https://golangci-lint.run/)
- [gofumpt](https://github.com/mvdan/gofumpt)
- [Conventional Commits](https://www.conventionalcommits.org/)
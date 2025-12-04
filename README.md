# API Gateway Template

A transparent HTTP gateway for proxying requests to backend services with JWT authentication, CORS, and structured logging.

## Features

- **Transparent Proxying** - forward HTTP requests to backend services without modification
- **JWT Authentication** - full JWT implementation with token validation and role-based access control
- **Multi-Backend Support** - route to multiple backend services by path prefix
- **CORS** - configurable cross-origin request policy
- **Structured Logging** - Uber Zap with JSON/console output
- **Graceful Shutdown** - proper shutdown with waiting for active requests
- **Health Check** - endpoint for monitoring gateway status
- **Chi Router** - lightweight and idiomatic router following stdlib standards

## Quick Start

### 1. Clone and Setup

#### Clone and Rename

```bash
# Clone the template
git clone <this-repo> my-gateway
cd my-gateway

# Rename the Go module
make rename MODULE=github.com/mycompany/my-gateway
```

#### Install Git Hooks (Recommended)

Install pre-commit hooks to automatically check code quality:

```bash
make install-hooks
```

This will run `go fmt`, `go vet`, and `go test` before each commit.

### 2. Configuration

Copy the example configuration:

```bash
cp .env.example .env
```

Edit `.env`:

```bash
# Backend services (choose one option)

# Option 1: Single backend (legacy)
PROXY_TARGET_URL=http://localhost:9000

# Option 2: Multiple backends (recommended)
CRM_SERVICE_URL=http://localhost:9001
CBS_SERVICE_URL=http://localhost:9002
BILLING_SERVICE_URL=http://localhost:9003

# JWT secret (CHANGE THIS IN PRODUCTION!)
JWT_SECRET=your-secret-key-change-this-in-production
```

### 3. Run

```bash
# Development
make run

# Or build and run
make build
./bin/api-gateway

# Docker
docker-compose up
```

### 4. Test

```bash
# Health check
curl http://localhost:8080/health

# Request (single backend)
curl http://localhost:8080/api/users

# Request to specific service (multi-backend)
curl http://localhost:8080/crm/api/customers
```

## Architecture

```
┌─────────┐      ┌──────────────┐      ┌──────────┐
│ Client  │─────▶│  API Gateway │─────▶│ Backend  │
│         │◀─────│              │◀─────│ Service  │
└─────────┘      └──────────────┘      └──────────┘
                       │
                       │ Middleware:
                       ├─ Logging (Zap)
                       ├─ CORS
                       └─ JWT Auth
```

### Operation Modes

**Single backend (legacy)**:

- All requests go to one backend
- `PROXY_TARGET_URL=http://backend:9000`
- Client URL: `http://gateway:8080/any/path`

**Multi-backend**:

- Route by path prefix
- `CRM_SERVICE_URL=http://crm:9001`
- Client URL: `http://gateway:8080/crm/any/path` → `http://crm:9001/any/path`

**Adding a new service:**

1. Add environment variable `YOUR_SERVICE_URL` to `.env`:
   ```bash
   YOUR_SERVICE_URL=http://your-service:9004
   ```

2. Add service name to `internal/config/config.go` in `loadProxyTargets()`:
   ```go
   serviceNames := []string{"CRM", "CBS", "BILLING", "AUTH", "NOTIFICATION", "PAYMENT", "YOUR"}
   ```

3. Restart gateway - service will be available at `http://gateway:8080/your/*`

## Main Commands

```bash
make help              # Show all commands
make build             # Build binary
make run               # Run in dev mode
make test              # Run tests
make fmt               # Format code
make vet               # Static analysis
make clean             # Clean build artifacts
```

## Project Structure

```
.
├── cmd/api/              # Application entry point
│   └── main.go          # Chi router setup, middleware, routing
├── internal/             # Internal code (not exported)
│   ├── config/          # Configuration from env
│   ├── middleware/      # Chi HTTP middleware
│   │   └── chi_middleware.go  # Logging, CORS, Auth
│   └── proxy/           # Reverse proxy logic
│       ├── proxy.go     # ReverseProxy implementation
│       └── factory.go   # Multi-backend proxy factory
├── pkg/                  # Public packages (exported)
│   ├── auth/            # JWT authentication
│   │   ├── jwt.go       # Token generation, validation, refresh
│   │   └── middleware.go  # Auth utilities, RBAC helpers
│   └── logger/          # Zap logger
│       ├── logger.go    # ZapLogger implementation
│       ├── interface.go # Logger interface
│       └── mock.go      # Mock logger for tests
├── docs/                 # Documentation
├── .env.example         # Example configuration
├── Dockerfile           # Docker image
├── docker-compose.yml   # Docker Compose for development
└── Makefile             # Build commands
```

## Technology Stack

- **Router**: [Chi v5](https://github.com/go-chi/chi) - lightweight, idiomatic HTTP router
- **Logger**: [Uber Zap](https://github.com/uber-go/zap) - blazing fast, structured logging
- **JWT**: [golang-jwt/jwt](https://github.com/golang-jwt/jwt) - JWT implementation for Go
- **Reverse Proxy**: `net/http/httputil.ReverseProxy` - standard library reverse proxy

## Documentation

- [Configuration](docs/CONFIGURATION.md) - all environment parameters
- [Deployment](docs/DEPLOYMENT.md) - production deployment guide

## Requirements

- Go 1.23+
- (optional) Docker and Docker Compose

## Security Notes

⚠️ **IMPORTANT**: The included JWT authentication is a LOCAL IMPLEMENTATION for development/testing only!

Before deploying to production, you MUST:
1. Change `JWT_SECRET` to a strong, randomly generated secret
2. Consider replacing the local auth with your corporate authentication middleware
3. Review and adjust CORS settings for your domain
4. Enable HTTPS/TLS in production

## License

MIT

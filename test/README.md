# Integration Tests

This directory contains integration tests for the API Gateway.

## Structure

```
test/
├── mock-backend/           # Mock backend service for testing
│   ├── main.go            # Simple HTTP server that simulates backend services
│   └── Dockerfile         # Docker image for mock backend
├── integration/           # Integration test suite
│   ├── gateway_test.go    # Main integration tests
│   └── helpers.go         # Test helper functions
├── docker-compose.test.yml # Docker Compose for test environment
└── README.md              # This file
```

## Running Integration Tests Locally

### Prerequisites

- Docker and Docker Compose
- Go 1.23+

### Run tests

```bash
# Start test environment
docker compose -f test/docker-compose.test.yml up -d

# Wait for services to be ready
sleep 10

# Run integration tests
go test -v -tags=integration ./test/integration/...

# Stop test environment
docker compose -f test/docker-compose.test.yml down
```

### Or use one command

```bash
# Start, test, and cleanup
docker compose -f test/docker-compose.test.yml up -d && \
  sleep 10 && \
  go test -v -tags=integration ./test/integration/... && \
  docker compose -f test/docker-compose.test.yml down
```

## What Integration Tests Check

The integration tests verify:

1. **Gateway Health Check** - `/health` endpoint availability
2. **Service Routing** - Requests are correctly routed to backend services:
   - `/crm/*` → CRM service
   - `/cbs/*` → CBS service
   - `/billing/*` → Billing service
3. **JWT Authentication** - Token validation and authorization:
   - Requests without token are rejected (401)
   - Valid tokens are accepted
   - Invalid tokens are rejected
   - User ID is extracted and passed to backend
4. **CORS Headers** - Cross-origin requests are handled correctly
5. **Proxy Behavior** - Gateway correctly proxies requests and responses
6. **Error Handling** - Backend errors are properly forwarded

## Mock Backend

The mock backend (`test/mock-backend/`) provides:

- `/health` - Health check endpoint
- `/api/echo` - Returns request information (method, path, headers)
- `/api/users` - Simulates user data endpoint (GET/POST)
- `/api/protected` - Checks for Authorization header
- `/api/error` - Returns 500 error for testing error handling

## Test Environment

The test environment (`docker-compose.test.yml`) includes:

- **mock-crm** - CRM service mock (port 9001)
- **mock-cbs** - CBS service mock (port 9002)
- **mock-billing** - Billing service mock (port 9003)
- **api-gateway** - API Gateway (port 8080)

All services are connected via `test-network` bridge network.

## CI/CD Integration

Integration tests are automatically run in GitLab CI pipeline:

```yaml
# .gitlab-ci.yml
integration-test:
  stage: integration-test
  script:
    - docker compose -f test/docker-compose.test.yml up -d
    - sleep 10
    - go test -v -tags=integration ./test/integration/...
  after_script:
    - docker compose -f test/docker-compose.test.yml down
```

## Build Tags

Integration tests use the `integration` build tag to separate them from unit tests:

```go
//go:build integration
// +build integration
```

This allows you to run unit tests without starting the test environment:

```bash
# Run only unit tests (fast)
go test ./...

# Run integration tests (requires test environment)
go test -tags=integration ./test/integration/...
```

## Troubleshooting

### Services not starting

Check Docker logs:
```bash
docker compose -f test/docker-compose.test.yml logs
```

### Connection refused errors

Make sure services are healthy:
```bash
docker compose -f test/docker-compose.test.yml ps
```

All services should show "healthy" status.

### Tests timing out

Increase wait time in `TestMain`:
```go
waitForService(gatewayURL+"/health", 60*time.Second)
```

Or check if port 8080 is already in use:
```bash
lsof -i :8080
```
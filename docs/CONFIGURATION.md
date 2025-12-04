# Configuration

All gateway settings are configured through environment variables. On startup, the application attempts to load the `.env` file if it exists.

## Environment Variables

### Server

| Variable | Description | Default Value |
|----------|-------------|---------------|
| `SERVER_HOST` | IP address to listen on | `0.0.0.0` |
| `SERVER_PORT` | Port to listen on | `8080` |
| `SERVER_READ_TIMEOUT` | Request read timeout | `15s` |
| `SERVER_WRITE_TIMEOUT` | Response write timeout | `15s` |
| `SERVER_IDLE_TIMEOUT` | Idle connection timeout | `60s` |

**Example:**
```bash
SERVER_HOST=127.0.0.1
SERVER_PORT=3000
SERVER_READ_TIMEOUT=30s
```

### CORS

| Variable | Description | Default Value |
|----------|-------------|---------------|
| `CORS_ALLOWED_ORIGINS` | Allowed origins (comma-separated) | `*` |
| `CORS_ALLOWED_METHODS` | Allowed HTTP methods | `GET,POST,PUT,DELETE,OPTIONS,PATCH` |
| `CORS_ALLOWED_HEADERS` | Allowed headers | `Content-Type,Authorization` |
| `CORS_ALLOW_CREDENTIALS` | Allow credentials | `true` |
| `CORS_MAX_AGE` | Preflight request cache (seconds) | `3600` |

**Example:**
```bash
CORS_ALLOWED_ORIGINS=http://localhost:3000,https://app.example.com
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE
CORS_ALLOWED_HEADERS=Content-Type,Authorization,X-Request-ID
```

### JWT Authentication

| Variable | Description | Default Value |
|----------|-------------|---------------|
| `JWT_SECRET` | Secret key for signing JWT tokens | **REQUIRED** |
| `JWT_ISSUER` | Token issuer | `api-gateway` |
| `JWT_AUDIENCE` | Token audience | `api-gateway` |
| `JWT_EXPIRATION` | Token expiration duration | `24h` |

**Example:**
```bash
JWT_SECRET=your-super-secret-key-min-32-chars-long
JWT_ISSUER=my-api-gateway
JWT_AUDIENCE=my-api-gateway
JWT_EXPIRATION=1h
```

⚠️ **SECURITY**:
- `JWT_SECRET` MUST be changed in production
- Use a strong, randomly generated secret (minimum 32 characters)
- Never commit secrets to version control

### Proxy (Backend Services)

#### Option 1: Single Backend (legacy)

| Variable | Description |
|----------|-------------|
| `PROXY_TARGET_URL` | URL of the single backend service |

**Example:**
```bash
PROXY_TARGET_URL=http://backend:9000
```

In this mode, all requests are proxied to one backend:
- `GET http://gateway:8080/api/users` → `GET http://backend:9000/api/users`

#### Option 2: Multi-Backend (recommended)

| Variable | Description |
|----------|-------------|
| `CRM_SERVICE_URL` | CRM service URL |
| `CBS_SERVICE_URL` | CBS service URL |
| `BILLING_SERVICE_URL` | Billing service URL |
| `AUTH_SERVICE_URL` | Authentication service URL |
| `NOTIFICATION_SERVICE_URL` | Notification service URL |
| `PAYMENT_SERVICE_URL` | Payment service URL |

**Example:**
```bash
CRM_SERVICE_URL=http://crm-service:9001
CBS_SERVICE_URL=http://cbs-service:9002
BILLING_SERVICE_URL=http://billing-service:9003
```

In this mode, requests are routed by prefix:
- `GET http://gateway:8080/crm/api/customers` → `GET http://crm-service:9001/api/customers`
- `POST http://gateway:8080/billing/invoices` → `POST http://billing-service:9003/invoices`

**Note:** The service prefix (`/crm`, `/billing`) is stripped before proxying.

#### General Proxy Settings

| Variable | Description | Default Value |
|----------|-------------|---------------|
| `PROXY_TIMEOUT` | Backend request timeout | `30s` |

**Example:**
```bash
PROXY_TIMEOUT=60s
```

### Logging

| Variable | Description | Default Value |
|----------|-------------|---------------|
| `LOG_LEVEL` | Log level (`debug`, `info`, `warn`, `error`) | `info` |
| `LOG_COMPONENT_NAME` | Component name in logs | `api-gateway` |

**Example for production:**
```bash
LOG_LEVEL=info
LOG_COMPONENT_NAME=api-gateway-prod
```

**Example for development:**
```bash
LOG_LEVEL=debug
LOG_COMPONENT_NAME=api-gateway-dev
```

**Log Output:**
- In production mode (`LOG_LEVEL=info`): JSON format to stdout
- In development mode (`LOG_LEVEL=debug`): Colorized console format
- Structured logging with fields: timestamp, level, message, component, and custom fields

## Complete Configuration Examples

### Development (.env)

```bash
# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000
CORS_ALLOW_CREDENTIALS=true

# JWT
JWT_SECRET=dev-secret-key-change-in-production-min-32-chars
JWT_EXPIRATION=24h

# Proxy - multi-backend
CRM_SERVICE_URL=http://localhost:9001
CBS_SERVICE_URL=http://localhost:9002
PROXY_TIMEOUT=30s

# Logging
LOG_LEVEL=debug
LOG_COMPONENT_NAME=api-gateway-dev
```

### Production (.env)

```bash
# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_READ_TIMEOUT=30s
SERVER_WRITE_TIMEOUT=30s

# CORS
CORS_ALLOWED_ORIGINS=https://app.example.com,https://admin.example.com
CORS_ALLOW_CREDENTIALS=true

# JWT (use strong secrets in production!)
JWT_SECRET=${JWT_SECRET}  # Load from secure secret store
JWT_ISSUER=api-gateway-prod
JWT_AUDIENCE=api-gateway-prod
JWT_EXPIRATION=1h

# Proxy - multi-backend
CRM_SERVICE_URL=http://crm-service:9001
CBS_SERVICE_URL=http://cbs-service:9002
BILLING_SERVICE_URL=http://billing-service:9003
PROXY_TIMEOUT=60s

# Logging
LOG_LEVEL=info
LOG_COMPONENT_NAME=api-gateway-prod
```

## Validation

On startup, the application validates required parameters:

- At least one backend must be configured (`PROXY_TARGET_URL` or `*_SERVICE_URL`)
- `JWT_SECRET` must be set and non-empty
- `SERVER_PORT` must be in range 1-65535
- Backend URLs must be valid

If validation fails, the application exits with an error.

## Testing Configuration

To test without authentication (useful for development):

```bash
export SKIP_AUTH=true
```

⚠️ **WARNING**: Never use `SKIP_AUTH=true` in production!

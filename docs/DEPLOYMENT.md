# Deployment

Guide for deploying the API Gateway in various environments.

## Docker

### Build Image

```bash
docker build -t api-gateway:latest .
```

### Run Container

```bash
docker run -d \
  --name api-gateway \
  -p 8080:8080 \
  -e JWT_SECRET=your-production-secret-min-32-chars-long \
  -e PROXY_TARGET_URL=http://backend:9000 \
  -e LOG_LEVEL=info \
  api-gateway:latest
```

**Multi-backend example:**

```bash
docker run -d \
  --name api-gateway \
  -p 8080:8080 \
  -e JWT_SECRET=your-production-secret-min-32-chars-long \
  -e CRM_SERVICE_URL=http://crm:9001 \
  -e CBS_SERVICE_URL=http://cbs:9002 \
  -e LOG_LEVEL=info \
  api-gateway:latest
```

## Docker Compose

### Basic Configuration

```yaml
version: '3.8'

services:
  gateway:
    image: api-gateway:latest
    ports:
      - "8080:8080"
    environment:
      - SERVER_PORT=8080
      - JWT_SECRET=your-production-secret-min-32-chars-long
      - JWT_ISSUER=api-gateway-prod
      - JWT_AUDIENCE=api-gateway-prod
      - JWT_EXPIRATION=1h
      - CRM_SERVICE_URL=http://crm:9001
      - CBS_SERVICE_URL=http://cbs:9002
      - LOG_LEVEL=info
      - LOG_COMPONENT_NAME=api-gateway
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
```

### Run

```bash
docker-compose up -d
```

## Kubernetes

### Secret for JWT

First, create a secret for JWT:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: gateway-secrets
type: Opaque
stringData:
  jwt-secret: your-super-secret-production-key-min-32-chars
```

Apply:
```bash
kubectl apply -f secret.yaml
```

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-gateway
  labels:
    app: api-gateway
spec:
  replicas: 3
  selector:
    matchLabels:
      app: api-gateway
  template:
    metadata:
      labels:
        app: api-gateway
    spec:
      containers:
      - name: gateway
        image: api-gateway:latest
        ports:
        - containerPort: 8080
          name: http
          protocol: TCP
        env:
        - name: SERVER_PORT
          value: "8080"
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: gateway-secrets
              key: jwt-secret
        - name: JWT_ISSUER
          value: "api-gateway-prod"
        - name: JWT_AUDIENCE
          value: "api-gateway-prod"
        - name: JWT_EXPIRATION
          value: "1h"
        - name: CRM_SERVICE_URL
          value: "http://crm-service:9001"
        - name: CBS_SERVICE_URL
          value: "http://cbs-service:9002"
        - name: LOG_LEVEL
          value: "info"
        - name: LOG_COMPONENT_NAME
          value: "api-gateway"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
          timeoutSeconds: 3
          failureThreshold: 3
        resources:
          requests:
            memory: "64Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "500m"
```

### Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: api-gateway
  labels:
    app: api-gateway
spec:
  selector:
    app: api-gateway
  ports:
  - port: 80
    targetPort: 8080
    protocol: TCP
    name: http
  type: LoadBalancer
```

### ConfigMap for Environment Variables

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: gateway-config
data:
  SERVER_PORT: "8080"
  LOG_LEVEL: "info"
  LOG_COMPONENT_NAME: "api-gateway"
  CRM_SERVICE_URL: "http://crm-service:9001"
  CBS_SERVICE_URL: "http://cbs-service:9002"
  PROXY_TIMEOUT: "60s"
  JWT_ISSUER: "api-gateway-prod"
  JWT_AUDIENCE: "api-gateway-prod"
  JWT_EXPIRATION: "1h"
```

Use in Deployment:

```yaml
envFrom:
- configMapRef:
    name: gateway-config
env:
- name: JWT_SECRET
  valueFrom:
    secretKeyRef:
      name: gateway-secrets
      key: jwt-secret
```

### Apply All Resources

```bash
kubectl apply -f secret.yaml
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
```

## Systemd (Bare Metal)

### Build Binary

```bash
make build
```

### Create Systemd Unit

`/etc/systemd/system/api-gateway.service`:

```ini
[Unit]
Description=API Gateway
After=network.target

[Service]
Type=simple
User=gateway
Group=gateway
WorkingDirectory=/opt/api-gateway
EnvironmentFile=/opt/api-gateway/.env
ExecStart=/opt/api-gateway/bin/api-gateway
Restart=always
RestartSec=10

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/api-gateway

[Install]
WantedBy=multi-user.target
```

### Installation

```bash
# Create user
sudo useradd -r -s /bin/false gateway

# Create directory
sudo mkdir -p /opt/api-gateway/bin
sudo cp bin/api-gateway /opt/api-gateway/bin/
sudo cp .env /opt/api-gateway/

# Set permissions
sudo chown -R gateway:gateway /opt/api-gateway
sudo chmod 600 /opt/api-gateway/.env
sudo chmod 755 /opt/api-gateway/bin/api-gateway

# Start service
sudo systemctl daemon-reload
sudo systemctl enable api-gateway
sudo systemctl start api-gateway

# Check status
sudo systemctl status api-gateway

# View logs
sudo journalctl -u api-gateway -f
```

## Reverse Proxy (Nginx)

### Basic Configuration

```nginx
upstream api_gateway {
    server 127.0.0.1:8080;

    # For multiple instances
    # server 127.0.0.1:8081;
    # server 127.0.0.1:8082;
}

server {
    listen 80;
    server_name api.example.com;

    # Redirect to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name api.example.com;

    # SSL certificates
    ssl_certificate /etc/letsencrypt/live/api.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.example.com/privkey.pem;

    # SSL settings
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "DENY" always;

    location / {
        proxy_pass http://api_gateway;
        proxy_http_version 1.1;

        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Host $host;

        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;

        # Buffering
        proxy_buffering off;
        proxy_request_buffering off;
    }

    location /health {
        access_log off;
        proxy_pass http://api_gateway;
    }
}
```

## Monitoring

### Health Check

```bash
curl http://localhost:8080/health
```

Response: `OK` (HTTP 200)

### Logs

Logs are output to stdout in JSON format (production) or console format (development).

**Production log example (JSON):**
```json
{
  "timestamp": "2025-11-20T12:00:00.000Z",
  "level": "info",
  "message": "http request processed",
  "component": "api-gateway",
  "client_ip": "192.168.1.1",
  "method": "GET",
  "path": "/api/users",
  "status": 200,
  "latency_ms": 45,
  "user_id": "user-123"
}
```

**Development log example (console):**
```
2025-11-20T12:00:00.000Z  INFO  http request processed
  component: api-gateway
  client_ip: 192.168.1.1
  method: GET
  path: /api/users
  status: 200
  latency_ms: 45
```

### Collect Logs

**Docker:**
```bash
docker logs -f api-gateway
```

**Kubernetes:**
```bash
kubectl logs -f deployment/api-gateway
```

**Systemd:**
```bash
journalctl -u api-gateway -f
```

**Forward to log aggregation service:**
- Use stdout and capture with your logging infrastructure (Fluentd, Logstash, etc.)
- Structured JSON format makes parsing easy

## Security Best Practices

### Production Checklist

- [ ] Use strong `JWT_SECRET` (minimum 32 random characters)
- [ ] Store secrets in secure secret management (Kubernetes Secrets, Vault, etc.)
- [ ] Enable HTTPS/TLS (use Let's Encrypt or corporate certificates)
- [ ] Configure proper CORS origins (never use `*` in production)
- [ ] Set appropriate timeouts
- [ ] Enable resource limits in Kubernetes
- [ ] Use non-root user in containers
- [ ] Keep dependencies updated
- [ ] Monitor logs for security events
- [ ] Implement rate limiting (via nginx or ingress)

### Environment-Specific Settings

**Development:**
```bash
LOG_LEVEL=debug
JWT_EXPIRATION=24h
CORS_ALLOWED_ORIGINS=http://localhost:3000
```

**Staging:**
```bash
LOG_LEVEL=info
JWT_EXPIRATION=1h
CORS_ALLOWED_ORIGINS=https://staging.example.com
```

**Production:**
```bash
LOG_LEVEL=info
JWT_EXPIRATION=1h
CORS_ALLOWED_ORIGINS=https://app.example.com,https://admin.example.com
```

## Troubleshooting

### Gateway not starting

1. Check configuration validation:
```bash
# Run locally to see validation errors
./bin/api-gateway
```

2. Verify required variables are set:
- `JWT_SECRET` must be non-empty
- At least one backend URL must be configured
- `SERVER_PORT` must be valid (1-65535)

### Authentication failures

1. Verify JWT configuration matches token generation
2. Check token expiration
3. Ensure `JWT_SECRET` is same across all instances
4. Review logs for specific error messages

### Backend connection issues

1. Verify backend URLs are reachable from gateway
2. Check network policies in Kubernetes
3. Verify timeout settings
4. Review proxy error logs

### Performance issues

1. Check resource limits (CPU/Memory)
2. Increase replica count for horizontal scaling
3. Adjust timeout settings
4. Monitor log output for bottlenecks

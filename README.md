# Reverse Payment Proxy Server

A production-ready Go-based reverse proxy server for SEPA QR code micro-payments following PSD2 regulations. This system enables content creators to accept micro-payments with minimal fees by leveraging direct bank-to-bank transfers through QR codes.

## 🚀 Features

- **Reverse Proxy Architecture**: Seamless integration with existing websites
- **SEPA QR Code Payments**: Mobile-friendly payment experience
- **Real-time Bank Monitoring**: Automated payment detection and processing
- **Multi-tenant Support**: Multiple merchants on single instance
- **PSD2 Compliance**: Leveraging European banking regulations
- **Instant Settlement**: Sub-15-second payment confirmation
- **Comprehensive API**: REST endpoints for all operations
- **High Performance**: PostgreSQL + Redis caching architecture
- **Production Ready**: Docker, monitoring, logging, and error handling

## 📋 Prerequisites

- Go 1.21+
- PostgreSQL 12+
- Redis 6+
- Make (optional, for convenience commands)

## 🛠 Installation

### Method 1: Local Development

1. **Clone and setup:**
```bash
git clone <repository-url>
cd micro-payments
make dev-setup
```

2. **Configure database connection:**
```bash
# Edit config/config.yaml with your database settings
vim config/config.yaml
```

3. **Run the server:**
```bash
make run
# or for hot reload during development:
make dev
```

### Method 2: Docker Compose (Recommended)

```bash
# Start all services (PostgreSQL, Redis, API)
make docker-run

# Or manually:
docker-compose up --build
```

### Method 3: Production Docker

```bash
make docker-build
docker run -p 8080:8080 \
  -e DATABASE_HOST=your-db-host \
  -e DATABASE_PASSWORD=your-password \
  payment-proxy:latest
```

## 🔧 Configuration

The server uses `config/config.yaml` for configuration. Key settings:

```yaml
server:
  port: 8080                    # HTTP server port
  
database:
  host: "localhost"             # PostgreSQL host
  name: "payments"              # Database name
  user: "postgres"              # Database user
  
payment:
  session_ttl: 15m              # Payment session timeout
  default_currency: "EUR"       # Default currency
  
bank:
  sync_interval: 5s             # Transaction sync frequency
  
logging:
  level: "info"                 # Log level (debug, info, warn, error)
```

Environment variables override config file values.

## 📚 API Usage

### Authentication

All API endpoints require authentication using API keys:

```bash
curl -H "Authorization: Bearer demo_api_key_12345" \
  http://localhost:8080/api/v1/payment/session
```

### Create Payment Session

```bash
curl -X POST http://localhost:8080/api/v1/payment/session \
  -H "Authorization: Bearer demo_api_key_12345" \
  -H "Content-Type: application/json" \
  -d '{
    "content_path": "/premium/article",
    "user_identifier": "user_12345",
    "return_url": "https://example.com/premium/article"
  }'
```

Response includes QR code data for payment:
```json
{
  "session_id": "uuid",
  "payment_reference": "PAY123...",
  "qr_code_svg": "<svg>...</svg>",
  "amount": 2.50,
  "currency": "EUR",
  "expires_at": "2024-01-01T12:00:00Z"
}
```

### Check Payment Status

```bash
curl http://localhost:8080/api/v1/payment/session/{session_id}/status
```

### Check Content Access

```bash
curl "http://localhost:8080/api/v1/content/{content_id}/access?user_identifier=user_12345"
```

## 🏗 Architecture

### System Components

```
┌─────────────────┐    ┌───────────────────┐    ┌─────────────────┐
│   User Browser  │────│  Reverse Proxy    │────│  Content Server │
└─────────────────┘    └───────────────────┘    └─────────────────┘
                                 │
                       ┌─────────┴─────────┐
                       │   Payment API     │
                       └─────────┬─────────┘
                                 │
                       ┌─────────┴─────────┐
                       │   Bank Monitor    │
                       └─────────┬─────────┘
                                 │
                       ┌─────────┴─────────┐
                       │ PostgreSQL + Redis│
                       └───────────────────┘
```

### Database Schema

The system implements a comprehensive schema with:
- **Merchants**: Content providers and their bank accounts
- **Content**: Protected content with pricing rules
- **Payment Sessions**: Individual payment attempts
- **Bank Transactions**: Detected bank transfers
- **Content Access**: Granted access permissions

### Payment Flow

1. **User Access**: User requests protected content
2. **Payment Check**: System checks existing access
3. **QR Generation**: Creates SEPA QR code for payment
4. **Bank Transfer**: User pays via banking app
5. **Transaction Detection**: System detects incoming payment
6. **Access Grant**: Immediate content access granted

## 🔒 Security

### Features

- **API Key Authentication**: Secure merchant authentication
- **Data Encryption**: Sensitive data encrypted at rest
- **Input Validation**: Comprehensive request validation
- **Rate Limiting**: Protection against abuse
- **Audit Logging**: Complete transaction audit trail
- **CORS Protection**: Secure cross-origin requests

### Production Considerations

- Use strong JWT secrets in production
- Enable HTTPS/TLS termination
- Configure proper firewall rules
- Set up monitoring and alerting
- Regular security updates

## 🧪 Testing

### Run Tests
```bash
make test
```

### Create Test Payment
```bash
# Start server and create a test payment session
make test-payment
```

### Health Check
```bash
make health
# or
curl http://localhost:8080/health
```

## 📊 Monitoring

### Health Endpoints

- `GET /health` - Service health status
- `GET /metrics` - Prometheus metrics (if enabled)

### Logging

Structured JSON logging with configurable levels:
```json
{
  "level": "info",
  "timestamp": "2024-01-01T12:00:00Z",
  "message": "Payment session created",
  "merchant_id": "uuid",
  "session_id": "uuid",
  "amount_cents": 250
}
```

## 🚀 Deployment

### Environment Variables

Key environment variables for production:

```bash
DATABASE_HOST=your-db-host
DATABASE_PASSWORD=your-secure-password
REDIS_ADDR=your-redis-host:6379
JWT_SECRET=your-super-secure-jwt-secret
LOG_LEVEL=info
```

### Docker Deployment

```bash
# Build production image
make docker-build

# Deploy with environment file
docker run --env-file .env -p 8080:8080 payment-proxy:latest
```

### Kubernetes Deployment

See `k8s/` directory for Kubernetes manifests (if available).

## 🧩 Integration

### Reverse Proxy Integration

The system acts as a reverse proxy. Configure your domain to point to this service:

```nginx
# Nginx upstream configuration
upstream payment-proxy {
    server 127.0.0.1:8080;
}

server {
    listen 443 ssl;
    server_name your-domain.com;
    
    location / {
        proxy_pass http://payment-proxy;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Webhook Integration

Configure webhooks for real-time payment notifications:

```yaml
merchant:
  webhook_url: "https://your-domain.com/webhook/payment"
  webhook_secret: "your-webhook-secret"
```

## 🛠 Development

### Hot Reload Development
```bash
make dev
```

### Code Formatting
```bash
make fmt
```

### Linting
```bash
make lint
```

### Generate Documentation
```bash
make docs
```

## 📖 Documentation

- **API Documentation**: Visit `/swagger/` when server is running
- **Database Schema**: See `docs/data_model.md`
- **Architecture**: See `docs/implementation_model.md`
- **Project Overview**: See `docs/project_brief.md`

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/your-feature`
3. Make changes and add tests
4. Run tests: `make test`
5. Format code: `make fmt`
6. Commit changes: `git commit -am 'Add feature'`
7. Push to branch: `git push origin feature/your-feature`
8. Submit a Pull Request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🆘 Support

- **Issues**: [GitHub Issues](https://github.com/your-repo/issues)
- **Documentation**: Check the `docs/` directory
- **Examples**: See `examples/` directory for integration examples

## 🎯 Roadmap

- [ ] Bank API integrations for major EU banks
- [ ] Subscription payment support
- [ ] Refund processing
- [ ] Advanced analytics dashboard
- [ ] White-label deployment options
- [ ] Mobile SDK for easy integration
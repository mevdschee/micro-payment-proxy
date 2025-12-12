# Reverse Payment Proxy - Implementation Model

## System Architecture Overview

This document outlines the technical implementation strategy for the Reverse Payment Proxy system, including architecture, components, APIs, deployment, and operational considerations.

## High-Level Architecture

```mermaid
graph TB
    subgraph "User Layer"
        U[User Browser]
        M[Mobile Banking App]
    end
    
    subgraph "Edge Layer"
        LB[Load Balancer]
        CDN[CDN/Cache]
    end
    
    subgraph "Application Layer"
        RP[Reverse Proxy]
        API[Payment API]
        WEB[Admin Dashboard]
        QR[QR Generator Service]
    end
    
    subgraph "Core Services"
        PS[Payment Service]
        BM[Bank Monitor]
        NS[Notification Service]
        AS[Auth Service]
    end
    
    subgraph "Data Layer"
        DB[(PostgreSQL)]
        CACHE[(Redis)]
        MQ[Message Queue]
    end
    
    subgraph "External"
        BANK[Bank APIs]
        CONTENT[Content Servers]
    end
    
    U --> LB
    M --> QR
    LB --> CDN
    CDN --> RP
    RP --> API
    RP --> CONTENT
    API --> PS
    PS --> BM
    BM --> BANK
    PS --> NS
    NS --> MQ
    PS --> DB
    BM --> DB
    API --> CACHE
    API --> AS
```

## Component Specifications

### 1. Reverse Proxy Service

**Technology**: Nginx with custom Lua modules or Go-based proxy

**Responsibilities**:
- Intercept requests for protected content
- Check payment status in real-time
- Serve payment pages or protected content
- Handle SSL termination and security headers

**Implementation**:
```go
// Core proxy handler
type ProxyHandler struct {
    paymentService *PaymentService
    contentCache   *ContentCache
    config         *ProxyConfig
}

func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 1. Parse request and identify merchant
    merchant := h.identifyMerchant(r.Host)
    
    // 2. Check if content requires payment
    content := h.getContentConfig(merchant.ID, r.URL.Path)
    if content == nil {
        h.proxyToOrigin(w, r)
        return
    }
    
    // 3. Check existing access
    userID := h.getUserIdentifier(r)
    access := h.paymentService.CheckAccess(userID, content.ID)
    
    if access.IsValid() {
        h.proxyToOrigin(w, r)
        h.logAccess(access, r)
        return
    }
    
    // 4. Serve payment page
    h.servePaymentPage(w, r, content)
}
```

**Configuration**:
```nginx
# Nginx configuration for high-performance proxying
upstream payment_api {
    server 127.0.0.1:8080;
    keepalive 32;
}

server {
    listen 443 ssl http2;
    server_name ~^(?<merchant>.+)\.payments\.domain\.com$;
    
    location /api/payment {
        proxy_pass http://payment_api;
        proxy_cache_bypass $http_pragma;
    }
    
    location / {
        access_by_lua_block {
            local payment = require "payment_check"
            payment.check_access()
        }
        
        proxy_pass http://backend_servers;
    }
}
```

### 2. Payment Service API

**Technology**: Go with Gin framework or Node.js with Express

**Core Endpoints**:

```yaml
# OpenAPI 3.0 Specification
openapi: 3.0.0
info:
  title: Reverse Payment Proxy API
  version: 1.0.0

paths:
  /api/v1/payment/session:
    post:
      summary: Create payment session
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                content_path:
                  type: string
                  example: "/premium/article/123"
                user_identifier:
                  type: string
                  example: "user_12345_or_ip_address"
                return_url:
                  type: string
                  example: "https://example.com/premium/article/123"
      responses:
        '201':
          description: Payment session created
          content:
            application/json:
              schema:
                type: object
                properties:
                  session_id:
                    type: string
                    format: uuid
                  payment_reference:
                    type: string
                    example: "PAY123456789012345"
                  qr_code_svg:
                    type: string
                  qr_code_data:
                    type: string
                  amount:
                    type: number
                    example: 2.50
                  currency:
                    type: string
                    example: "EUR"
                  expires_at:
                    type: string
                    format: date-time

  /api/v1/payment/session/{session_id}/status:
    get:
      summary: Check payment status
      parameters:
        - name: session_id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        '200':
          description: Payment status
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string
                    enum: [pending, paid, expired, failed]
                  paid_at:
                    type: string
                    format: date-time
                  access_expires_at:
                    type: string
                    format: date-time

  /api/v1/content/{content_id}/access:
    get:
      summary: Check content access
      parameters:
        - name: content_id
          in: path
          required: true
          schema:
            type: string
            format: uuid
        - name: user_identifier
          in: query
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Access status
          content:
            application/json:
              schema:
                type: object
                properties:
                  has_access:
                    type: boolean
                  expires_at:
                    type: string
                    format: date-time
                  remaining_time:
                    type: integer
                    description: Seconds remaining
```

**Service Implementation**:
```go
type PaymentService struct {
    db           *sql.DB
    qrGenerator  *QRGenerator
    bankMonitor  *BankMonitor
    cache        *redis.Client
}

func (s *PaymentService) CreatePaymentSession(req CreateSessionRequest) (*PaymentSession, error) {
    // 1. Validate content and pricing
    content, err := s.getContent(req.MerchantID, req.ContentPath)
    if err != nil {
        return nil, err
    }
    
    // 2. Generate unique payment reference
    paymentRef := s.generatePaymentReference()
    
    // 3. Create QR code with SEPA format
    qrData := s.generateSEPAQRCode(content.PriceCents, paymentRef, content.MerchantIBAN)
    
    // 4. Store session in database
    session := &PaymentSession{
        ID:               uuid.New(),
        MerchantID:       req.MerchantID,
        ContentID:        content.ID,
        UserIdentifier:   req.UserIdentifier,
        AmountCents:      content.PriceCents,
        PaymentReference: paymentRef,
        QRCodeData:       qrData,
        Status:           "pending",
        ExpiresAt:        time.Now().Add(15 * time.Minute),
    }
    
    err = s.storeSession(session)
    if err != nil {
        return nil, err
    }
    
    // 5. Schedule cleanup job
    s.scheduleCleanup(session.ID, session.ExpiresAt)
    
    return session, nil
}
```

### 3. Bank Monitoring Service

**Technology**: Go with concurrent processing

**Responsibilities**:
- Poll bank APIs for new transactions
- Match transactions to payment sessions
- Update payment status in real-time
- Handle API rate limits and errors

**Implementation**:
```go
type BankMonitor struct {
    connections []*BankConnection
    processor   *TransactionProcessor
    scheduler   *cron.Cron
}

func (bm *BankMonitor) StartMonitoring() {
    // Monitor each bank connection
    for _, conn := range bm.connections {
        go bm.monitorConnection(conn)
    }
    
    // Schedule periodic full sync
    bm.scheduler.AddFunc("@every 30s", bm.syncAllAccounts)
    bm.scheduler.Start()
}

func (bm *BankMonitor) monitorConnection(conn *BankConnection) {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            transactions, err := bm.fetchNewTransactions(conn)
            if err != nil {
                log.Error("Failed to fetch transactions", "error", err, "connection", conn.ID)
                continue
            }
            
            for _, tx := range transactions {
                go bm.processTransaction(tx, conn)
            }
            
        case <-bm.ctx.Done():
            return
        }
    }
}

func (bm *BankMonitor) processTransaction(tx *BankTransaction, conn *BankConnection) {
    // 1. Extract payment reference from transaction
    paymentRef := bm.extractPaymentReference(tx.RemittanceInfo)
    if paymentRef == "" {
        return // Not a payment proxy transaction
    }
    
    // 2. Find matching payment session
    session, err := bm.findPaymentSession(paymentRef)
    if err != nil {
        log.Warn("No matching session for payment reference", "ref", paymentRef)
        return
    }
    
    // 3. Validate transaction amount and currency
    if !bm.validateTransaction(tx, session) {
        log.Warn("Transaction validation failed", "tx", tx.ID, "session", session.ID)
        return
    }
    
    // 4. Update payment status
    err = bm.markSessionPaid(session, tx)
    if err != nil {
        log.Error("Failed to mark session as paid", "error", err)
        return
    }
    
    // 5. Grant content access
    bm.grantContentAccess(session)
    
    // 6. Send webhook notification
    bm.sendWebhookNotification(session, tx)
}
```

### 4. QR Code Generator Service

**Technology**: Go with QR code libraries

**SEPA QR Code Format**:
```go
type SEPAQRGenerator struct {
    template string
}

func (g *SEPAQRGenerator) GenerateQRCode(amount int, reference, iban, bic string) (string, error) {
    // BCD format for SEPA QR codes
    data := fmt.Sprintf(
        "BCD\n002\n1\nSCT\n%s\n%s\n%s\nEUR%.2f\n\n%s\n",
        bic,
        "Payment Proxy Service", // Merchant name
        iban,
        float64(amount)/100,
        reference,
    )
    
    qr, err := qrcode.New(data, qrcode.Medium)
    if err != nil {
        return "", err
    }
    
    // Return as SVG for web display
    svg := qr.ToSVG(256)
    return svg, nil
}
```

### 5. Admin Dashboard

**Technology**: React with TypeScript, Material-UI

**Key Features**:
- Merchant onboarding and management
- Real-time payment monitoring
- Analytics and reporting
- Content configuration
- Bank connection setup

**Component Structure**:
```typescript
// Dashboard main component
interface DashboardProps {
    merchantId: string;
}

const Dashboard: React.FC<DashboardProps> = ({ merchantId }) => {
    const [metrics, setMetrics] = useState<MetricsData>();
    const [payments, setPayments] = useState<Payment[]>();
    
    useEffect(() => {
        // Real-time updates via WebSocket
        const ws = new WebSocket(`wss://api.payments.com/merchants/${merchantId}/live`);
        ws.onmessage = (event) => {
            const update = JSON.parse(event.data);
            handleRealtimeUpdate(update);
        };
    }, [merchantId]);
    
    return (
        <Grid container spacing={3}>
            <Grid item xs={12} md={3}>
                <MetricCard
                    title="Today's Revenue"
                    value={formatCurrency(metrics?.todayRevenue)}
                    change={metrics?.revenueChange}
                />
            </Grid>
            <Grid item xs={12} md={3}>
                <MetricCard
                    title="Active Sessions"
                    value={metrics?.activeSessions}
                    change={metrics?.sessionsChange}
                />
            </Grid>
            <Grid item xs={12}>
                <PaymentTable payments={payments} />
            </Grid>
        </Grid>
    );
};
```

## Database Implementation

### Primary Database: PostgreSQL

**Configuration**:
```sql
-- Performance optimization
shared_preload_libraries = 'pg_stat_statements'
max_connections = 200
shared_buffers = 256MB
effective_cache_size = 1GB
random_page_cost = 1.1

-- Connection pooling with PgBouncer
[databases]
payments_db = host=localhost port=5432 dbname=payments_db

[pgbouncer]
pool_mode = transaction
max_client_conn = 100
default_pool_size = 25
```

### Cache Layer: Redis

**Configuration**:
```redis
# Performance settings
maxmemory 1gb
maxmemory-policy allkeys-lru
timeout 300
tcp-keepalive 300

# Persistence for durability
save 900 1
save 300 10
save 60 10000
```

**Caching Strategy**:
```go
type CacheService struct {
    redis *redis.Client
}

func (c *CacheService) GetPaymentSession(sessionID string) (*PaymentSession, error) {
    // Try cache first
    cached, err := c.redis.Get(fmt.Sprintf("session:%s", sessionID)).Result()
    if err == nil {
        var session PaymentSession
        json.Unmarshal([]byte(cached), &session)
        return &session, nil
    }
    
    // Fallback to database
    session, err := c.db.GetPaymentSession(sessionID)
    if err != nil {
        return nil, err
    }
    
    // Cache for future requests
    data, _ := json.Marshal(session)
    c.redis.Set(fmt.Sprintf("session:%s", sessionID), data, 15*time.Minute)
    
    return session, nil
}
```

## Security Implementation

### 1. Authentication & Authorization

**JWT-based API Authentication**:
```go
type AuthService struct {
    jwtSecret []byte
}

func (a *AuthService) GenerateToken(merchantID string, scopes []string) (string, error) {
    claims := jwt.MapClaims{
        "merchant_id": merchantID,
        "scopes":      scopes,
        "exp":         time.Now().Add(24 * time.Hour).Unix(),
        "iat":         time.Now().Unix(),
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(a.jwtSecret)
}

func (a *AuthService) ValidateToken(tokenString string) (*Claims, error) {
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        return a.jwtSecret, nil
    })
    
    if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
        return &Claims{
            MerchantID: claims["merchant_id"].(string),
            Scopes:     claims["scopes"].([]string),
        }, nil
    }
    
    return nil, err
}
```

### 2. Data Encryption

**Sensitive Data Encryption**:
```go
type EncryptionService struct {
    key []byte
}

func (e *EncryptionService) EncryptSensitiveData(data string) (string, error) {
    block, err := aes.NewCipher(e.key)
    if err != nil {
        return "", err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    nonce := make([]byte, gcm.NonceSize())
    if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }
    
    ciphertext := gcm.Seal(nonce, nonce, []byte(data), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}
```

### 3. Rate Limiting

**API Rate Limiting**:
```go
func RateLimitMiddleware() gin.HandlerFunc {
    limiter := rate.NewLimiter(rate.Every(time.Second), 10) // 10 requests per second
    
    return func(c *gin.Context) {
        if !limiter.Allow() {
            c.JSON(429, gin.H{"error": "Rate limit exceeded"})
            c.Abort()
            return
        }
        c.Next()
    }
}
```

## Deployment Architecture

### 1. Container Configuration

**Dockerfile for Payment API**:
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o payment-api ./cmd/api

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/payment-api .
COPY --from=builder /app/configs ./configs
EXPOSE 8080
CMD ["./payment-api"]
```

**Docker Compose for Development**:
```yaml
version: '3.8'
services:
  payment-api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://user:pass@postgres:5432/payments
      - REDIS_URL=redis://redis:6379
    depends_on:
      - postgres
      - redis

  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: payments
      POSTGRES_USER: user
      POSTGRES_PASSWORD: pass
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/nginx/ssl

volumes:
  postgres_data:
  redis_data:
```

### 2. Kubernetes Deployment

**Payment API Deployment**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: payment-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: payment-api
  template:
    metadata:
      labels:
        app: payment-api
    spec:
      containers:
      - name: payment-api
        image: payment-proxy/api:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: url
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5

---
apiVersion: v1
kind: Service
metadata:
  name: payment-api-service
spec:
  selector:
    app: payment-api
  ports:
  - port: 80
    targetPort: 8080
  type: ClusterIP
```

### 3. Infrastructure as Code (Terraform)

**AWS Infrastructure**:
```hcl
# VPC and networking
resource "aws_vpc" "main" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true
  
  tags = {
    Name = "payment-proxy-vpc"
  }
}

# EKS Cluster
resource "aws_eks_cluster" "payment_cluster" {
  name     = "payment-proxy"
  role_arn = aws_iam_role.cluster.arn
  version  = "1.28"

  vpc_config {
    subnet_ids = [aws_subnet.private[*].id, aws_subnet.public[*].id]
    endpoint_private_access = true
    endpoint_public_access  = true
  }

  depends_on = [
    aws_iam_role_policy_attachment.cluster_AmazonEKSClusterPolicy,
  ]
}

# RDS Instance
resource "aws_db_instance" "payments" {
  identifier     = "payment-proxy-db"
  engine         = "postgres"
  engine_version = "15.4"
  instance_class = "db.t3.medium"
  
  allocated_storage     = 100
  max_allocated_storage = 1000
  storage_encrypted     = true
  
  db_name  = "payments"
  username = var.db_username
  password = var.db_password
  
  vpc_security_group_ids = [aws_security_group.rds.id]
  db_subnet_group_name   = aws_db_subnet_group.main.name
  
  backup_retention_period = 7
  backup_window          = "03:00-04:00"
  maintenance_window     = "sun:04:00-sun:05:00"
  
  skip_final_snapshot = false
  final_snapshot_identifier = "payment-proxy-final-snapshot"
  
  tags = {
    Name = "payment-proxy-db"
  }
}

# ElastiCache Redis
resource "aws_elasticache_subnet_group" "main" {
  name       = "payment-proxy-cache-subnet"
  subnet_ids = aws_subnet.private[*].id
}

resource "aws_elasticache_replication_group" "redis" {
  replication_group_id       = "payment-proxy-redis"
  description                = "Redis cluster for payment proxy"
  
  port                = 6379
  parameter_group_name = "default.redis7"
  
  num_cache_clusters = 2
  node_type         = "cache.t3.micro"
  
  subnet_group_name  = aws_elasticache_subnet_group.main.name
  security_group_ids = [aws_security_group.redis.id]
  
  at_rest_encryption_enabled = true
  transit_encryption_enabled = true
  
  tags = {
    Name = "payment-proxy-redis"
  }
}
```

## Monitoring & Observability

### 1. Metrics Collection

**Prometheus Configuration**:
```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "payment_proxy_rules.yml"

scrape_configs:
  - job_name: 'payment-api'
    static_configs:
      - targets: ['payment-api:8080']
    metrics_path: /metrics
    scrape_interval: 5s

  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres-exporter:9187']

  - job_name: 'redis'
    static_configs:
      - targets: ['redis-exporter:9121']

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093
```

**Custom Metrics in Go**:
```go
var (
    paymentSessionsCreated = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "payment_sessions_created_total",
            Help: "Total number of payment sessions created",
        },
        []string{"merchant_id", "content_type"},
    )
    
    paymentProcessingDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "payment_processing_duration_seconds",
            Help:    "Time taken to process payments",
            Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30},
        },
        []string{"status"},
    )
    
    activePaymentSessions = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "payment_sessions_active",
            Help: "Number of active payment sessions",
        },
        []string{"merchant_id"},
    )
)

func init() {
    prometheus.MustRegister(paymentSessionsCreated)
    prometheus.MustRegister(paymentProcessingDuration)
    prometheus.MustRegister(activePaymentSessions)
}
```

### 2. Logging Strategy

**Structured Logging with Zap**:
```go
func setupLogger() *zap.Logger {
    config := zap.NewProductionConfig()
    config.OutputPaths = []string{"stdout"}
    config.ErrorOutputPaths = []string{"stderr"}
    
    logger, _ := config.Build()
    return logger
}

func (s *PaymentService) CreateSession(ctx context.Context, req *CreateSessionRequest) {
    logger := s.logger.With(
        zap.String("merchant_id", req.MerchantID),
        zap.String("content_path", req.ContentPath),
        zap.String("user_id", req.UserIdentifier),
    )
    
    logger.Info("Creating payment session")
    
    session, err := s.createPaymentSession(req)
    if err != nil {
        logger.Error("Failed to create payment session", zap.Error(err))
        return
    }
    
    logger.Info("Payment session created successfully",
        zap.String("session_id", session.ID),
        zap.String("payment_reference", session.PaymentReference),
        zap.Time("expires_at", session.ExpiresAt),
    )
}
```

### 3. Health Checks

**Comprehensive Health Check Endpoint**:
```go
type HealthChecker struct {
    db    *sql.DB
    redis *redis.Client
}

func (h *HealthChecker) Check(c *gin.Context) {
    status := map[string]interface{}{
        "status":    "healthy",
        "timestamp": time.Now().Unix(),
        "checks":    make(map[string]interface{}),
    }
    
    // Database health
    if err := h.db.Ping(); err != nil {
        status["checks"]["database"] = map[string]interface{}{
            "status": "unhealthy",
            "error":  err.Error(),
        }
        status["status"] = "unhealthy"
    } else {
        status["checks"]["database"] = map[string]interface{}{
            "status": "healthy",
        }
    }
    
    // Redis health
    if err := h.redis.Ping().Err(); err != nil {
        status["checks"]["redis"] = map[string]interface{}{
            "status": "unhealthy",
            "error":  err.Error(),
        }
        status["status"] = "unhealthy"
    } else {
        status["checks"]["redis"] = map[string]interface{}{
            "status": "healthy",
        }
    }
    
    if status["status"] == "healthy" {
        c.JSON(200, status)
    } else {
        c.JSON(503, status)
    }
}
```

## Testing Strategy

### 1. Unit Tests

```go
func TestPaymentService_CreateSession(t *testing.T) {
    // Setup test database and dependencies
    db := setupTestDB(t)
    defer db.Close()
    
    service := &PaymentService{
        db:          db,
        qrGenerator: &MockQRGenerator{},
    }
    
    tests := []struct {
        name    string
        request CreateSessionRequest
        wantErr bool
    }{
        {
            name: "valid session creation",
            request: CreateSessionRequest{
                MerchantID:     "merchant-123",
                ContentPath:    "/premium/article",
                UserIdentifier: "user-456",
            },
            wantErr: false,
        },
        {
            name: "invalid merchant",
            request: CreateSessionRequest{
                MerchantID:     "invalid-merchant",
                ContentPath:    "/premium/article",
                UserIdentifier: "user-456",
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            session, err := service.CreateSession(tt.request)
            if (err != nil) != tt.wantErr {
                t.Errorf("CreateSession() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if !tt.wantErr {
                assert.NotEmpty(t, session.ID)
                assert.NotEmpty(t, session.PaymentReference)
                assert.Equal(t, "pending", session.Status)
            }
        })
    }
}
```

### 2. Integration Tests

```go
func TestPaymentFlow_EndToEnd(t *testing.T) {
    // Start test server
    server := httptest.NewServer(setupTestAPI())
    defer server.Close()
    
    client := &http.Client{}
    
    // 1. Create payment session
    sessionResp, err := createPaymentSession(client, server.URL)
    require.NoError(t, err)
    
    // 2. Simulate bank transaction
    tx := &BankTransaction{
        PaymentReference: sessionResp.PaymentReference,
        AmountCents:      sessionResp.AmountCents,
        Currency:         "EUR",
    }
    
    err = simulateBankTransaction(tx)
    require.NoError(t, err)
    
    // 3. Wait for processing
    time.Sleep(2 * time.Second)
    
    // 4. Check payment status
    status, err := checkPaymentStatus(client, server.URL, sessionResp.SessionID)
    require.NoError(t, err)
    assert.Equal(t, "paid", status.Status)
    
    // 5. Verify content access
    access, err := checkContentAccess(client, server.URL, sessionResp.ContentID)
    require.NoError(t, err)
    assert.True(t, access.HasAccess)
}
```

## Performance Optimization

### 1. Database Optimization

```sql
-- Optimized queries with proper indexing
EXPLAIN ANALYZE SELECT * FROM payment_sessions 
WHERE payment_reference = $1 AND status = 'pending';

-- Connection pooling configuration
ALTER SYSTEM SET max_connections = 200;
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
```

### 2. Caching Strategy

```go
// Multi-level caching
type CacheStrategy struct {
    l1 *freecache.Cache  // In-memory L1 cache
    l2 *redis.Client     // Redis L2 cache
    l3 *sql.DB          // Database L3
}

func (c *CacheStrategy) GetPaymentSession(sessionID string) (*PaymentSession, error) {
    // Try L1 cache (fastest)
    if data, err := c.l1.Get([]byte(sessionID)); err == nil {
        var session PaymentSession
        json.Unmarshal(data, &session)
        return &session, nil
    }
    
    // Try L2 cache (Redis)
    if data, err := c.l2.Get(sessionID).Result(); err == nil {
        var session PaymentSession
        json.Unmarshal([]byte(data), &session)
        
        // Populate L1 cache
        sessionData, _ := json.Marshal(&session)
        c.l1.Set([]byte(sessionID), sessionData, 300) // 5 minute TTL
        
        return &session, nil
    }
    
    // Fallback to database
    session, err := c.getSessionFromDB(sessionID)
    if err != nil {
        return nil, err
    }
    
    // Populate both cache levels
    sessionData, _ := json.Marshal(session)
    c.l1.Set([]byte(sessionID), sessionData, 300)
    c.l2.Set(sessionID, sessionData, 15*time.Minute)
    
    return session, nil
}
```

## Compliance & Legal

### 1. GDPR Compliance

```go
// Data retention and deletion
func (s *DataService) ProcessGDPRRequest(userID string, requestType GDPRRequestType) error {
    switch requestType {
    case GDPRRequestTypeExport:
        return s.exportUserData(userID)
    case GDPRRequestTypeDelete:
        return s.deleteUserData(userID)
    case GDPRRequestTypeCorrect:
        return s.correctUserData(userID)
    }
    return nil
}

func (s *DataService) deleteUserData(userID string) error {
    // Anonymize payment sessions
    _, err := s.db.Exec(`
        UPDATE payment_sessions 
        SET user_identifier = 'DELETED_USER',
            ip_address = NULL,
            user_agent = NULL
        WHERE user_identifier = $1
    `, userID)
    
    return err
}
```

### 2. PSD2 Compliance

```go
// Strong Customer Authentication validation
func (s *AuthService) ValidateSCA(transaction *Transaction) error {
    // Implement SCA validation logic
    // This would integrate with bank SCA APIs
    
    if transaction.Amount > 30*100 { // €30 limit
        return s.requireSCA(transaction)
    }
    
    return nil
}
```

This implementation model provides a comprehensive technical foundation for building the reverse payment proxy system with proper architecture, security, performance, and compliance considerations.
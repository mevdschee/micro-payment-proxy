-- Create database and extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_crypto";

-- Create custom types
CREATE TYPE merchant_status AS ENUM (
    'pending',
    'active', 
    'suspended',
    'deactivated'
);

CREATE TYPE content_type_enum AS ENUM (
    'webpage',
    'api_endpoint',
    'file_download',
    'streaming_media',
    'subscription'
);

CREATE TYPE payment_status AS ENUM (
    'pending',
    'paid',
    'expired',
    'cancelled',
    'failed',
    'refunded'
);

CREATE TYPE transaction_status AS ENUM (
    'detected',
    'matched',
    'processed',
    'ignored',
    'disputed'
);

CREATE TYPE sync_status_enum AS ENUM (
    'active',
    'error',
    'disconnected',
    'rate_limited'
);

CREATE TYPE log_severity AS ENUM ('debug', 'info', 'warning', 'error', 'critical');

-- Create tables
CREATE TABLE merchants (
    merchant_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    domain VARCHAR(255) NOT NULL,
    bank_account_iban VARCHAR(34) NOT NULL,
    bank_account_bic VARCHAR(11),
    webhook_url VARCHAR(500),
    webhook_secret VARCHAR(255),
    api_key VARCHAR(255) UNIQUE NOT NULL,
    status merchant_status DEFAULT 'pending',
    pricing_tier VARCHAR(50) DEFAULT 'basic',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    last_active_at TIMESTAMPTZ,
    settings JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}'
);

CREATE TABLE content (
    content_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id UUID REFERENCES merchants(merchant_id) ON DELETE CASCADE,
    path VARCHAR(1000) NOT NULL,
    title VARCHAR(255),
    description TEXT,
    price_cents INTEGER NOT NULL,
    currency VARCHAR(3) DEFAULT 'EUR',
    access_duration_seconds INTEGER DEFAULT 3600,
    content_type content_type_enum DEFAULT 'webpage',
    access_rules JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(merchant_id, path)
);

CREATE TABLE payment_sessions (
    session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id UUID REFERENCES merchants(merchant_id) ON DELETE CASCADE,
    content_id UUID REFERENCES content(content_id) ON DELETE CASCADE,
    user_identifier VARCHAR(255),
    amount_cents INTEGER NOT NULL,
    currency VARCHAR(3) DEFAULT 'EUR',
    payment_reference VARCHAR(35) UNIQUE NOT NULL,
    qr_code_data TEXT NOT NULL,
    status payment_status DEFAULT 'pending',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    paid_at TIMESTAMPTZ,
    access_granted_at TIMESTAMPTZ,
    access_expires_at TIMESTAMPTZ,
    user_agent TEXT,
    ip_address INET,
    metadata JSONB DEFAULT '{}'
);

CREATE TABLE bank_transactions (
    transaction_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id UUID REFERENCES merchants(merchant_id),
    bank_reference VARCHAR(100),
    payment_reference VARCHAR(35),
    amount_cents INTEGER NOT NULL,
    currency VARCHAR(3) NOT NULL,
    debtor_name VARCHAR(255),
    debtor_iban VARCHAR(34),
    creditor_iban VARCHAR(34) NOT NULL,
    transaction_date TIMESTAMPTZ NOT NULL,
    booking_date TIMESTAMPTZ NOT NULL,
    value_date TIMESTAMPTZ,
    status transaction_status DEFAULT 'detected',
    processed_at TIMESTAMPTZ,
    raw_data JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE content_access (
    access_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID REFERENCES payment_sessions(session_id) ON DELETE CASCADE,
    merchant_id UUID REFERENCES merchants(merchant_id) ON DELETE CASCADE,
    content_id UUID REFERENCES content(content_id) ON DELETE CASCADE,
    user_identifier VARCHAR(255) NOT NULL,
    granted_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    last_accessed_at TIMESTAMPTZ,
    access_count INTEGER DEFAULT 0,
    ip_address INET,
    user_agent TEXT,
    is_active BOOLEAN DEFAULT TRUE
);

CREATE TABLE bank_connections (
    connection_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id UUID REFERENCES merchants(merchant_id) ON DELETE CASCADE,
    bank_name VARCHAR(255) NOT NULL,
    bank_bic VARCHAR(11),
    account_iban VARCHAR(34) NOT NULL,
    api_endpoint VARCHAR(500) NOT NULL,
    client_id VARCHAR(255) NOT NULL,
    client_secret_encrypted TEXT NOT NULL,
    access_token_encrypted TEXT,
    refresh_token_encrypted TEXT,
    token_expires_at TIMESTAMPTZ,
    last_sync_at TIMESTAMPTZ,
    sync_status sync_status_enum DEFAULT 'active',
    error_count INTEGER DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE audit_logs (
    log_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id UUID REFERENCES merchants(merchant_id),
    session_id UUID REFERENCES payment_sessions(session_id),
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB NOT NULL,
    ip_address INET,
    user_agent TEXT,
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    severity log_severity DEFAULT 'info'
);

CREATE TABLE system_config (
    config_key VARCHAR(100) PRIMARY KEY,
    config_value JSONB NOT NULL,
    description TEXT,
    is_sensitive BOOLEAN DEFAULT FALSE,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    updated_by VARCHAR(255)
);

-- Create indexes for performance
CREATE INDEX idx_payment_sessions_reference ON payment_sessions(payment_reference);
CREATE INDEX idx_payment_sessions_status_expires ON payment_sessions(status, expires_at);
CREATE INDEX idx_content_merchant_path ON content(merchant_id, path);
CREATE INDEX idx_content_access_user_expires ON content_access(user_identifier, expires_at);
CREATE INDEX idx_bank_transactions_reference ON bank_transactions(payment_reference);
CREATE INDEX idx_bank_transactions_merchant_date ON bank_transactions(merchant_id, transaction_date);
CREATE INDEX idx_payment_sessions_merchant_status ON payment_sessions(merchant_id, status);
CREATE INDEX idx_content_access_active ON content_access(is_active, expires_at) WHERE is_active = true;
CREATE INDEX idx_audit_logs_merchant_timestamp ON audit_logs(merchant_id, timestamp);
CREATE INDEX idx_audit_logs_event_type ON audit_logs(event_type);
CREATE INDEX idx_audit_logs_session_id ON audit_logs(session_id);

-- Add constraints
ALTER TABLE content ADD CONSTRAINT check_price_positive CHECK (price_cents > 0);
ALTER TABLE payment_sessions ADD CONSTRAINT check_amount_positive CHECK (amount_cents > 0);
ALTER TABLE merchants ADD CONSTRAINT check_iban_format CHECK (bank_account_iban ~ '^[A-Z]{2}[0-9]{2}[A-Z0-9]{4}[0-9]{7}([A-Z0-9]?){0,16}$');
ALTER TABLE payment_sessions ADD CONSTRAINT check_expires_future CHECK (expires_at > created_at);
ALTER TABLE content_access ADD CONSTRAINT check_access_expires_future CHECK (expires_at > granted_at);

-- Insert sample data
INSERT INTO merchants (name, email, domain, bank_account_iban, api_key, status) VALUES
('Demo Merchant', 'demo@example.com', 'demo.example.com', 'DE89370400440532013000', 'demo_api_key_12345', 'active');

INSERT INTO content (merchant_id, path, title, price_cents, access_duration_seconds) VALUES
((SELECT merchant_id FROM merchants WHERE email = 'demo@example.com'), '/premium/article', 'Premium Article', 250, 3600),
((SELECT merchant_id FROM merchants WHERE email = 'demo@example.com'), '/premium/video', 'Premium Video', 500, 7200);
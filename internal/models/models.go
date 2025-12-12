package models

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Merchant represents a merchant in the system
type Merchant struct {
	MerchantID      uuid.UUID              `json:"merchant_id" db:"merchant_id"`
	Name            string                 `json:"name" db:"name"`
	Email           string                 `json:"email" db:"email"`
	Domain          string                 `json:"domain" db:"domain"`
	BankAccountIBAN string                 `json:"bank_account_iban" db:"bank_account_iban"`
	BankAccountBIC  *string                `json:"bank_account_bic,omitempty" db:"bank_account_bic"`
	WebhookURL      *string                `json:"webhook_url,omitempty" db:"webhook_url"`
	WebhookSecret   *string                `json:"webhook_secret,omitempty" db:"webhook_secret"`
	APIKey          string                 `json:"api_key" db:"api_key"`
	Status          MerchantStatus         `json:"status" db:"status"`
	PricingTier     string                 `json:"pricing_tier" db:"pricing_tier"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`
	LastActiveAt    *time.Time             `json:"last_active_at,omitempty" db:"last_active_at"`
	Settings        map[string]interface{} `json:"settings" db:"settings"`
	Metadata        map[string]interface{} `json:"metadata" db:"metadata"`
}

// Content represents content that can be accessed via payment
type Content struct {
	ContentID             uuid.UUID              `json:"content_id" db:"content_id"`
	MerchantID            uuid.UUID              `json:"merchant_id" db:"merchant_id"`
	Path                  string                 `json:"path" db:"path"`
	Title                 *string                `json:"title,omitempty" db:"title"`
	Description           *string                `json:"description,omitempty" db:"description"`
	PriceCents            int                    `json:"price_cents" db:"price_cents"`
	Currency              string                 `json:"currency" db:"currency"`
	AccessDurationSeconds int                    `json:"access_duration_seconds" db:"access_duration_seconds"`
	ContentType           ContentType            `json:"content_type" db:"content_type"`
	AccessRules           map[string]interface{} `json:"access_rules" db:"access_rules"`
	IsActive              bool                   `json:"is_active" db:"is_active"`
	CreatedAt             time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at" db:"updated_at"`
}

// PaymentSession represents a payment session for accessing content
type PaymentSession struct {
	SessionID        uuid.UUID              `json:"session_id" db:"session_id"`
	MerchantID       uuid.UUID              `json:"merchant_id" db:"merchant_id"`
	ContentID        uuid.UUID              `json:"content_id" db:"content_id"`
	UserIdentifier   *string                `json:"user_identifier,omitempty" db:"user_identifier"`
	AmountCents      int                    `json:"amount_cents" db:"amount_cents"`
	Currency         string                 `json:"currency" db:"currency"`
	PaymentReference string                 `json:"payment_reference" db:"payment_reference"`
	QRCodeData       string                 `json:"qr_code_data" db:"qr_code_data"`
	Status           PaymentStatus          `json:"status" db:"status"`
	ExpiresAt        time.Time              `json:"expires_at" db:"expires_at"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	PaidAt           *time.Time             `json:"paid_at,omitempty" db:"paid_at"`
	AccessGrantedAt  *time.Time             `json:"access_granted_at,omitempty" db:"access_granted_at"`
	AccessExpiresAt  *time.Time             `json:"access_expires_at,omitempty" db:"access_expires_at"`
	UserAgent        *string                `json:"user_agent,omitempty" db:"user_agent"`
	IPAddress        *string                `json:"ip_address,omitempty" db:"ip_address"`
	Metadata         map[string]interface{} `json:"metadata" db:"metadata"`
}

// BankTransaction represents a transaction from bank API
type BankTransaction struct {
	TransactionID    uuid.UUID               `json:"transaction_id" db:"transaction_id"`
	MerchantID       *uuid.UUID              `json:"merchant_id,omitempty" db:"merchant_id"`
	BankReference    *string                 `json:"bank_reference,omitempty" db:"bank_reference"`
	PaymentReference *string                 `json:"payment_reference,omitempty" db:"payment_reference"`
	AmountCents      int                     `json:"amount_cents" db:"amount_cents"`
	Currency         string                  `json:"currency" db:"currency"`
	DebtorName       *string                 `json:"debtor_name,omitempty" db:"debtor_name"`
	DebtorIBAN       *string                 `json:"debtor_iban,omitempty" db:"debtor_iban"`
	CreditorIBAN     string                  `json:"creditor_iban" db:"creditor_iban"`
	TransactionDate  time.Time               `json:"transaction_date" db:"transaction_date"`
	BookingDate      time.Time               `json:"booking_date" db:"booking_date"`
	ValueDate        *time.Time              `json:"value_date,omitempty" db:"value_date"`
	Status           TransactionStatus       `json:"status" db:"status"`
	ProcessedAt      *time.Time              `json:"processed_at,omitempty" db:"processed_at"`
	RawData          *map[string]interface{} `json:"raw_data,omitempty" db:"raw_data"`
	CreatedAt        time.Time               `json:"created_at" db:"created_at"`
}

// ContentAccess represents access granted to content
type ContentAccess struct {
	AccessID       uuid.UUID  `json:"access_id" db:"access_id"`
	SessionID      uuid.UUID  `json:"session_id" db:"session_id"`
	MerchantID     uuid.UUID  `json:"merchant_id" db:"merchant_id"`
	ContentID      uuid.UUID  `json:"content_id" db:"content_id"`
	UserIdentifier string     `json:"user_identifier" db:"user_identifier"`
	GrantedAt      time.Time  `json:"granted_at" db:"granted_at"`
	ExpiresAt      time.Time  `json:"expires_at" db:"expires_at"`
	LastAccessedAt *time.Time `json:"last_accessed_at,omitempty" db:"last_accessed_at"`
	AccessCount    int        `json:"access_count" db:"access_count"`
	IPAddress      *string    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent      *string    `json:"user_agent,omitempty" db:"user_agent"`
	IsActive       bool       `json:"is_active" db:"is_active"`
}

// Enum types
type MerchantStatus string

const (
	MerchantStatusPending     MerchantStatus = "pending"
	MerchantStatusActive      MerchantStatus = "active"
	MerchantStatusSuspended   MerchantStatus = "suspended"
	MerchantStatusDeactivated MerchantStatus = "deactivated"
)

type ContentType string

const (
	ContentTypeWebpage        ContentType = "webpage"
	ContentTypeAPIEndpoint    ContentType = "api_endpoint"
	ContentTypeFileDownload   ContentType = "file_download"
	ContentTypeStreamingMedia ContentType = "streaming_media"
	ContentTypeSubscription   ContentType = "subscription"
)

type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusPaid      PaymentStatus = "paid"
	PaymentStatusExpired   PaymentStatus = "expired"
	PaymentStatusCancelled PaymentStatus = "cancelled"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)

type TransactionStatus string

const (
	TransactionStatusDetected  TransactionStatus = "detected"
	TransactionStatusMatched   TransactionStatus = "matched"
	TransactionStatusProcessed TransactionStatus = "processed"
	TransactionStatusIgnored   TransactionStatus = "ignored"
	TransactionStatusDisputed  TransactionStatus = "disputed"
)

// Implement Valuer and Scanner interfaces for custom types to work with database/sql

func (ms MerchantStatus) Value() (driver.Value, error) {
	return string(ms), nil
}

func (ms *MerchantStatus) Scan(value interface{}) error {
	if value == nil {
		*ms = ""
		return nil
	}
	if str, ok := value.(string); ok {
		*ms = MerchantStatus(str)
		return nil
	}
	return fmt.Errorf("cannot scan %T into MerchantStatus", value)
}

func (ct ContentType) Value() (driver.Value, error) {
	return string(ct), nil
}

func (ct *ContentType) Scan(value interface{}) error {
	if value == nil {
		*ct = ""
		return nil
	}
	if str, ok := value.(string); ok {
		*ct = ContentType(str)
		return nil
	}
	return fmt.Errorf("cannot scan %T into ContentType", value)
}

func (ps PaymentStatus) Value() (driver.Value, error) {
	return string(ps), nil
}

func (ps *PaymentStatus) Scan(value interface{}) error {
	if value == nil {
		*ps = ""
		return nil
	}
	if str, ok := value.(string); ok {
		*ps = PaymentStatus(str)
		return nil
	}
	return fmt.Errorf("cannot scan %T into PaymentStatus", value)
}

func (ts TransactionStatus) Value() (driver.Value, error) {
	return string(ts), nil
}

func (ts *TransactionStatus) Scan(value interface{}) error {
	if value == nil {
		*ts = ""
		return nil
	}
	if str, ok := value.(string); ok {
		*ts = TransactionStatus(str)
		return nil
	}
	return fmt.Errorf("cannot scan %T into TransactionStatus", value)
}

package services

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mh74hf/micro-payments/internal/config"
	"github.com/mh74hf/micro-payments/internal/models"
	"go.uber.org/zap"
)

// PaymentService handles payment-related operations
type PaymentService struct {
	db     *sql.DB
	config *config.Config
	logger *zap.Logger
}

// NewPaymentService creates a new payment service
func NewPaymentService(db *sql.DB, cfg *config.Config, logger *zap.Logger) *PaymentService {
	return &PaymentService{
		db:     db,
		config: cfg,
		logger: logger,
	}
}

// CreatePaymentSession creates a new payment session
func (s *PaymentService) CreatePaymentSession(merchantID, contentID uuid.UUID, userIdentifier string) (*models.PaymentSession, error) {
	// First, get the content details to determine price
	var content models.Content
	query := `
		SELECT content_id, merchant_id, path, price_cents, currency, access_duration_seconds, is_active
		FROM content 
		WHERE content_id = $1 AND merchant_id = $2 AND is_active = true`

	err := s.db.QueryRow(query, contentID, merchantID).Scan(
		&content.ContentID,
		&content.MerchantID,
		&content.Path,
		&content.PriceCents,
		&content.Currency,
		&content.AccessDurationSeconds,
		&content.IsActive,
	)
	if err != nil {
		return nil, fmt.Errorf("content not found: %w", err)
	}

	// Generate payment reference and QR code data
	paymentRef := fmt.Sprintf("PAY-%d", time.Now().Unix())
	qrCodeData := fmt.Sprintf("SEPA QR Code Data for %s - Amount: %.2f %s", paymentRef, float64(content.PriceCents)/100, content.Currency)

	// Create payment session
	session := &models.PaymentSession{
		SessionID:        uuid.New(),
		MerchantID:       merchantID,
		ContentID:        contentID,
		AmountCents:      content.PriceCents,
		Currency:         content.Currency,
		PaymentReference: paymentRef,
		QRCodeData:       qrCodeData,
		Status:           models.PaymentStatusPending,
		ExpiresAt:        time.Now().Add(s.config.Payment.SessionTimeout),
		CreatedAt:        time.Now(),
	}

	if userIdentifier != "" {
		session.UserIdentifier = &userIdentifier
	}

	// Insert into database
	insertQuery := `
		INSERT INTO payment_sessions (
			session_id, merchant_id, content_id, user_identifier, amount_cents, 
			currency, payment_reference, qr_code_data, status, expires_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err = s.db.Exec(insertQuery,
		session.SessionID,
		session.MerchantID,
		session.ContentID,
		session.UserIdentifier,
		session.AmountCents,
		session.Currency,
		session.PaymentReference,
		session.QRCodeData,
		session.Status,
		session.ExpiresAt,
		session.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment session: %w", err)
	}

	return session, nil
}

// GetPaymentSession retrieves a payment session by ID
func (s *PaymentService) GetPaymentSession(sessionID uuid.UUID) (*models.PaymentSession, error) {
	var session models.PaymentSession
	query := `
		SELECT session_id, merchant_id, content_id, user_identifier, amount_cents,
		       currency, payment_reference, qr_code_data, status, expires_at,
		       created_at, paid_at, access_granted_at, access_expires_at
		FROM payment_sessions 
		WHERE session_id = $1`

	err := s.db.QueryRow(query, sessionID).Scan(
		&session.SessionID,
		&session.MerchantID,
		&session.ContentID,
		&session.UserIdentifier,
		&session.AmountCents,
		&session.Currency,
		&session.PaymentReference,
		&session.QRCodeData,
		&session.Status,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.PaidAt,
		&session.AccessGrantedAt,
		&session.AccessExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("payment session not found: %w", err)
	}

	return &session, nil
}

// VerifyPayment simulates payment verification (in real implementation, this would check bank APIs)
func (s *PaymentService) VerifyPayment(sessionID uuid.UUID) error {
	// For demonstration purposes, we'll simulate a successful payment
	// In a real implementation, this would check the bank's API for the payment

	query := `
		UPDATE payment_sessions 
		SET status = $1, paid_at = $2, access_granted_at = $2, access_expires_at = $3
		WHERE session_id = $4 AND status = $5`

	accessExpiresAt := time.Now().Add(time.Hour) // Default 1 hour access

	_, err := s.db.Exec(query,
		models.PaymentStatusPaid,
		time.Now(),
		accessExpiresAt,
		sessionID,
		models.PaymentStatusPending,
	)
	if err != nil {
		return fmt.Errorf("failed to verify payment: %w", err)
	}

	return nil
}

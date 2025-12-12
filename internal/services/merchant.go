package services

import (
	"database/sql"
	"fmt"

	"github.com/mh74hf/micro-payments/internal/models"
	"go.uber.org/zap"
)

// MerchantService handles merchant-related operations
type MerchantService struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewMerchantService creates a new merchant service
func NewMerchantService(db *sql.DB, logger *zap.Logger) *MerchantService {
	return &MerchantService{
		db:     db,
		logger: logger,
	}
}

// GetMerchantByAPIKey retrieves a merchant by API key
func (s *MerchantService) GetMerchantByAPIKey(apiKey string) (*models.Merchant, error) {
	var merchant models.Merchant
	query := `
		SELECT merchant_id, name, email, domain, bank_account_iban, 
		       api_key, status, pricing_tier, created_at, updated_at
		FROM merchants 
		WHERE api_key = $1 AND status = 'active'`

	err := s.db.QueryRow(query, apiKey).Scan(
		&merchant.MerchantID,
		&merchant.Name,
		&merchant.Email,
		&merchant.Domain,
		&merchant.BankAccountIBAN,
		&merchant.APIKey,
		&merchant.Status,
		&merchant.PricingTier,
		&merchant.CreatedAt,
		&merchant.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("merchant not found: %w", err)
	}

	return &merchant, nil
}

// GetMerchantByDomain retrieves a merchant by domain
func (s *MerchantService) GetMerchantByDomain(domain string) (*models.Merchant, error) {
	var merchant models.Merchant
	query := `
		SELECT merchant_id, name, email, domain, bank_account_iban, 
		       api_key, status, pricing_tier, created_at, updated_at
		FROM merchants 
		WHERE domain = $1 AND status = 'active'`

	err := s.db.QueryRow(query, domain).Scan(
		&merchant.MerchantID,
		&merchant.Name,
		&merchant.Email,
		&merchant.Domain,
		&merchant.BankAccountIBAN,
		&merchant.APIKey,
		&merchant.Status,
		&merchant.PricingTier,
		&merchant.CreatedAt,
		&merchant.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("merchant not found: %w", err)
	}

	return &merchant, nil
}

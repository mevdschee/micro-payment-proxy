package services

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/mh74hf/micro-payments/internal/models"
	"go.uber.org/zap"
)

// ContentService handles content-related operations
type ContentService struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewContentService creates a new content service
func NewContentService(db *sql.DB, logger *zap.Logger) *ContentService {
	return &ContentService{
		db:     db,
		logger: logger,
	}
}

// GetContentByPath retrieves content by merchant ID and path
func (s *ContentService) GetContentByPath(merchantID uuid.UUID, path string) (*models.Content, error) {
	var content models.Content
	query := `
		SELECT content_id, merchant_id, path, title, description, 
		       price_cents, currency, access_duration_seconds, content_type, is_active,
		       created_at, updated_at
		FROM content 
		WHERE merchant_id = $1 AND path = $2 AND is_active = true`

	err := s.db.QueryRow(query, merchantID, path).Scan(
		&content.ContentID,
		&content.MerchantID,
		&content.Path,
		&content.Title,
		&content.Description,
		&content.PriceCents,
		&content.Currency,
		&content.AccessDurationSeconds,
		&content.ContentType,
		&content.IsActive,
		&content.CreatedAt,
		&content.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("content not found: %w", err)
	}

	return &content, nil
}

// CheckAccess verifies if a user has access to content
func (s *ContentService) CheckAccess(contentID uuid.UUID, userIdentifier string) (*models.ContentAccess, error) {
	var access models.ContentAccess
	query := `
		SELECT access_id, session_id, merchant_id, content_id, user_identifier,
		       granted_at, expires_at, last_accessed_at, access_count, is_active
		FROM content_access 
		WHERE content_id = $1 AND user_identifier = $2 AND is_active = true AND expires_at > NOW()`

	err := s.db.QueryRow(query, contentID, userIdentifier).Scan(
		&access.AccessID,
		&access.SessionID,
		&access.MerchantID,
		&access.ContentID,
		&access.UserIdentifier,
		&access.GrantedAt,
		&access.ExpiresAt,
		&access.LastAccessedAt,
		&access.AccessCount,
		&access.IsActive,
	)
	if err != nil {
		return nil, fmt.Errorf("access not found: %w", err)
	}

	return &access, nil
}

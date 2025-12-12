package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mh74hf/micro-payments/internal/services"
	"go.uber.org/zap"
)

// Handlers holds all HTTP handlers
type Handlers struct {
	paymentService  *services.PaymentService
	merchantService *services.MerchantService
	contentService  *services.ContentService
	logger          *zap.Logger
}

// NewHandlers creates a new handlers instance
func NewHandlers(
	paymentService *services.PaymentService,
	merchantService *services.MerchantService,
	contentService *services.ContentService,
	logger *zap.Logger,
) *Handlers {
	return &Handlers{
		paymentService:  paymentService,
		merchantService: merchantService,
		contentService:  contentService,
		logger:          logger,
	}
}

// CreatePayment creates a new payment session
func (h *Handlers) CreatePayment(c *gin.Context) {
	var req struct {
		ContentPath    string `json:"content_path" binding:"required"`
		UserIdentifier string `json:"user_identifier"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get merchant from domain or API key
	domain := c.GetHeader("X-Merchant-Domain")
	if domain == "" {
		// Extract domain from Host header
		host := c.GetHeader("Host")
		if host != "" {
			domain = strings.Split(host, ":")[0]
		}
	}

	merchant, err := h.merchantService.GetMerchantByDomain(domain)
	if err != nil {
		h.logger.Error("Failed to get merchant", zap.Error(err), zap.String("domain", domain))
		c.JSON(http.StatusNotFound, gin.H{"error": "Merchant not found"})
		return
	}

	// Get content
	content, err := h.contentService.GetContentByPath(merchant.MerchantID, req.ContentPath)
	if err != nil {
		h.logger.Error("Failed to get content", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Content not found"})
		return
	}

	// Create payment session
	session, err := h.paymentService.CreatePaymentSession(merchant.MerchantID, content.ContentID, req.UserIdentifier)
	if err != nil {
		h.logger.Error("Failed to create payment session", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment session"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"session_id":        session.SessionID,
		"payment_reference": session.PaymentReference,
		"qr_code_data":      session.QRCodeData,
		"amount_cents":      session.AmountCents,
		"currency":          session.Currency,
		"expires_at":        session.ExpiresAt,
		"status":            session.Status,
	})
}

// GetPaymentStatus retrieves payment session status
func (h *Handlers) GetPaymentStatus(c *gin.Context) {
	sessionIDStr := c.Param("sessionId")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	session, err := h.paymentService.GetPaymentSession(sessionID)
	if err != nil {
		h.logger.Error("Failed to get payment session", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment session not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id":        session.SessionID,
		"status":            session.Status,
		"amount_cents":      session.AmountCents,
		"currency":          session.Currency,
		"expires_at":        session.ExpiresAt,
		"paid_at":           session.PaidAt,
		"access_granted_at": session.AccessGrantedAt,
		"access_expires_at": session.AccessExpiresAt,
	})
}

// VerifyPayment verifies a payment (simulated for demo)
func (h *Handlers) VerifyPayment(c *gin.Context) {
	sessionIDStr := c.Param("sessionId")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	err = h.paymentService.VerifyPayment(sessionID)
	if err != nil {
		h.logger.Error("Failed to verify payment", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify payment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Payment verified successfully"})
}

// ServeContent serves protected content if payment is verified
func (h *Handlers) ServeContent(c *gin.Context) {
	path := c.Param("path")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Get merchant from domain
	domain := c.GetHeader("X-Merchant-Domain")
	if domain == "" {
		host := c.GetHeader("Host")
		if host != "" {
			domain = strings.Split(host, ":")[0]
		}
	}

	merchant, err := h.merchantService.GetMerchantByDomain(domain)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Merchant not found"})
		return
	}

	// Get content
	content, err := h.contentService.GetContentByPath(merchant.MerchantID, path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Content not found"})
		return
	}

	// Check if user has access (simplified - in real implementation, use session/JWT)
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = c.ClientIP() // Fallback to IP
	}

	access, err := h.contentService.CheckAccess(content.ContentID, userID)
	if err != nil || access == nil {
		// No access - return payment required response
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error":        "Payment required",
			"content_path": path,
			"price_cents":  content.PriceCents,
			"currency":     content.Currency,
		})
		return
	}

	// User has access - serve content
	c.JSON(http.StatusOK, gin.H{
		"message":     "Content access granted",
		"content":     content,
		"access_info": access,
	})
}

// Placeholder handlers for merchant management
func (h *Handlers) GetMerchants(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get merchants - not implemented"})
}

func (h *Handlers) CreateMerchant(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Create merchant - not implemented"})
}

func (h *Handlers) UpdateMerchant(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Update merchant - not implemented"})
}

func (h *Handlers) DeleteMerchant(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Delete merchant - not implemented"})
}

// Placeholder handlers for admin
func (h *Handlers) GetStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get stats - not implemented"})
}

func (h *Handlers) GetTransactions(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get transactions - not implemented"})
}

// ReverseProxy handles the main reverse proxy functionality
func (h *Handlers) ReverseProxy(c *gin.Context) {
	// This is a simplified reverse proxy implementation
	// In a real implementation, this would forward requests to the actual backend
	c.JSON(http.StatusOK, gin.H{
		"message": "Reverse proxy - content would be served from backend",
		"path":    c.Request.URL.Path,
		"method":  c.Request.Method,
	})
}

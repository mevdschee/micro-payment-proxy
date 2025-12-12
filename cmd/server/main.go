package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mh74hf/micro-payments/internal/config"
	"github.com/mh74hf/micro-payments/internal/database"
	"github.com/mh74hf/micro-payments/internal/handlers"
	"github.com/mh74hf/micro-payments/internal/middleware"
	"github.com/mh74hf/micro-payments/internal/services"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Initialize database
	db, err := database.NewConnection(cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize services
	paymentService := services.NewPaymentService(db, cfg, logger)
	merchantService := services.NewMerchantService(db, logger)
	contentService := services.NewContentService(db, logger)

	// Initialize handlers
	handlers := handlers.NewHandlers(paymentService, merchantService, contentService, logger)

	// Set up Gin router
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger(logger))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
		})
	})

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Payment routes
		payments := v1.Group("/payments")
		{
			payments.POST("/", handlers.CreatePayment)
			payments.GET("/:sessionId", handlers.GetPaymentStatus)
			payments.POST("/:sessionId/verify", handlers.VerifyPayment)
		}

		// Content access routes
		content := v1.Group("/content")
		{
			content.GET("/*path", handlers.ServeContent)
		}

		// Merchant routes (authenticated)
		merchants := v1.Group("/merchants")
		merchants.Use(middleware.AuthRequired())
		{
			merchants.GET("/", handlers.GetMerchants)
			merchants.POST("/", handlers.CreateMerchant)
			merchants.PUT("/:id", handlers.UpdateMerchant)
			merchants.DELETE("/:id", handlers.DeleteMerchant)
		}

		// Admin routes (authenticated)
		admin := v1.Group("/admin")
		admin.Use(middleware.AuthRequired())
		{
			admin.GET("/stats", handlers.GetStats)
			admin.GET("/transactions", handlers.GetTransactions)
		}
	}

	// Proxy routes - this handles the reverse proxy functionality
	router.NoRoute(handlers.ReverseProxy)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting server",
			zap.String("host", cfg.Server.Host),
			zap.Int("port", cfg.Server.Port),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

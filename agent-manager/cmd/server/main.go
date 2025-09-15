package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vot3k/agent-handoff/agent-manager/internal/config"
	"github.com/vot3k/agent-handoff/agent-manager/internal/handlers"
	"github.com/vot3k/agent-handoff/agent-manager/internal/middleware"
	"github.com/vot3k/agent-handoff/agent-manager/internal/repository"
	"github.com/vot3k/agent-handoff/agent-manager/internal/service"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize Redis client
	redisClient, err := repository.NewRedisClient(cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to initialize Redis client: %v", err)
	}
	defer redisClient.Close()

	// Initialize repositories
	handoffRepo := repository.NewHandoffRepository(redisClient)

	// Initialize services
	handoffService := service.NewHandoffService(handoffRepo, cfg)

	// Initialize handlers
	handoffHandler := handlers.NewHandoffHandler(handoffService)
	healthHandler := handlers.NewHealthHandler(redisClient)

	// Setup router with middleware
	router := setupRouter(handoffHandler, healthHandler)

	// Create HTTP server
	server := &http.Server{
		Addr:         cfg.Server.Address,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting HTTP server on %s", cfg.Server.Address)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server gracefully
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func setupRouter(handoffHandler *handlers.HandoffHandler, healthHandler *handlers.HealthHandler) http.Handler {
	mux := http.NewServeMux()

	// Health check endpoints
	mux.HandleFunc("/health", healthHandler.Health)
	mux.HandleFunc("/health/ready", healthHandler.Ready)

	// Handoff management endpoints
	mux.HandleFunc("POST /api/v1/handoffs", handoffHandler.CreateHandoff)
	mux.HandleFunc("GET /api/v1/handoffs/{id}", handoffHandler.GetHandoff)
	mux.HandleFunc("GET /api/v1/handoffs", handoffHandler.ListHandoffs)
	mux.HandleFunc("PUT /api/v1/handoffs/{id}/status", handoffHandler.UpdateStatus)

	// Queue management endpoints
	mux.HandleFunc("GET /api/v1/queues", handoffHandler.ListQueues)
	mux.HandleFunc("GET /api/v1/queues/{queue}/depth", handoffHandler.GetQueueDepth)

	// Apply middleware stack
	handler := middleware.Chain(
		mux,
		middleware.RequestID,
		middleware.Logger,
		middleware.CORS,
		middleware.Recovery,
		middleware.Timeout(30*time.Second),
		middleware.RateLimit(100), // 100 requests per minute
	)

	return handler
}

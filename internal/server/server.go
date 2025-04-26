package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/neekaru/whatsappgo-bot/internal/app"
	"github.com/neekaru/whatsappgo-bot/internal/config"
	"github.com/neekaru/whatsappgo-bot/pkg/logger"
)

// Server represents the HTTP server
type Server struct {
	router *gin.Engine
	app    *app.App
	config *config.Config
}

// NewServer creates a new server instance
func NewServer(app *app.App, config *config.Config) *Server {
	// Set up gin to log to the same log file
	gin.DefaultWriter = io.MultiWriter(os.Stdout, logger.GetWriter(app.Logger))
	gin.DefaultErrorWriter = io.MultiWriter(os.Stderr, logger.GetWriter(app.Logger))

	r := gin.Default()

	// Configure CORS
	corsConfig := config.GetCorsConfig()
	r.Use(cors.New(corsConfig))

	return &Server{
		router: r,
		app:    app,
		config: config,
	}
}

// Router returns the gin router
func (s *Server) Router() *gin.Engine {
	return s.router
}

// Start starts the HTTP server
func (s *Server) Start() error {
	srv := &http.Server{
		Addr:    ":" + s.config.ServerPort,
		Handler: s.router,
	}

	go func() {
		s.app.Logger.Printf("🚀 WhatsApp bot running on :%s", s.config.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.app.Logger.Printf("Server error: %v\n", err)
		}
	}()

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	srv := &http.Server{
		Addr:    ":" + s.config.ServerPort,
		Handler: s.router,
	}

	s.app.Logger.Println("🚫 Shutting down server...")
	if err := srv.Shutdown(ctx); err != nil {
		s.app.Logger.Printf("Server forced to shutdown: %v\n", err)
		return fmt.Errorf("server forced to shutdown: %v", err)
	}

	s.app.Logger.Println("Server exited")
	return nil
}

package server

import (
	"github.com/neekaru/whatsappgo-bot/internal/auth"
	"github.com/neekaru/whatsappgo-bot/internal/contact"
	"github.com/neekaru/whatsappgo-bot/internal/health"
	"github.com/neekaru/whatsappgo-bot/internal/media"
	"github.com/neekaru/whatsappgo-bot/internal/messaging"
	"github.com/neekaru/whatsappgo-bot/internal/session"
)

// SetupRoutes configures all the routes for the application
func (s *Server) SetupRoutes() {
	// Register health check handlers
	healthHandlers := health.NewHandlers(s.app)
	s.router.GET("/", healthHandlers.RootHandler)
	s.router.GET("/health", healthHandlers.HealthCheckHandler)
	s.router.GET("/health/", healthHandlers.HealthCheckHandlerWithSlash)

	// Register session handlers
	sessionHandlers := session.NewHandlers(s.app)
	s.router.POST("/wa/add", sessionHandlers.AddSessionHandler)
	s.router.POST("/wa/status", sessionHandlers.StatusHandler)
	s.router.GET("/wa/status", sessionHandlers.StatusHandler)
	s.router.POST("/wa/restart", sessionHandlers.RestartHandler)
	s.router.POST("/wa/logout", sessionHandlers.LogoutHandler)

	// Register authentication handlers
	authHandlers := auth.NewHandlers(s.app)
	s.router.GET("/wa/qr-image", authHandlers.QRImageHandler)

	// Register messaging handlers
	messagingHandlers := messaging.NewHandlers(s.app)
	s.router.POST("/send", messagingHandlers.SendMessageHandler)
	s.router.POST("/msg/read", messagingHandlers.MarkReadHandler)

	// Register media handlers
	mediaHandlers := media.NewHandlers(s.app)
	s.router.POST("/send/file", mediaHandlers.SendFileHandler)
	s.router.POST("/send/image", mediaHandlers.SendImageHandler)
	s.router.POST("/send/video", mediaHandlers.SendVideoHandler)

	// Register contact handlers
	contactHandlers := contact.NewHandlers(s.app)
	s.router.POST("/contact", contactHandlers.GetAllContactsHandler)
	s.router.POST("/contact/saved", contactHandlers.GetSavedContactsHandler)
	s.router.POST("/contact/unsaved", contactHandlers.GetUnsavedContactsHandler)
	s.router.POST("/contact/refresh", contactHandlers.RefreshContactsHandler)
}

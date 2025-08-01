package config

import (
	"os"
	"time"

	"github.com/gin-contrib/cors"
)

// Config holds application configuration
type Config struct {
	ServerPort string
	DataDir    string
}

// NewConfig creates a new configuration with default values
func NewConfig() *Config {
	return &Config{
		ServerPort: "3000",
		DataDir:    "data",
	}
}

// EnsureDataDir ensures the data directory exists
func (c *Config) EnsureDataDir() error {
	return os.MkdirAll(c.DataDir, 0755)
}

// GetCorsConfig returns CORS configuration for the application
func (c *Config) GetCorsConfig() cors.Config {
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"}
	corsConfig.ExposeHeaders = []string{"Content-Length", "Content-Type"}
	corsConfig.AllowCredentials = true
	corsConfig.MaxAge = 12 * time.Hour
	return corsConfig
}

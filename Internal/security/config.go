package security

import(
    "time"
)


// Config holds all security related configurations
type Config struct {
	JWTSecret           string
	AccessTokenDuration time.Duration
	RefreshTokenDuration time.Duration
	AllowedOrigins      []string
}

// NewConfig creates a new security configuration
func NewConfig() *Config {
	return &Config{
		JWTSecret:           "your-super-secret-key", // In production, use environment variables
		AccessTokenDuration: 15 * time.Minute,
		RefreshTokenDuration: 7 * 24 * time.Hour,
		AllowedOrigins:      []string{"http://localhost:3000", "https://yourdomain.com"},
	}
}
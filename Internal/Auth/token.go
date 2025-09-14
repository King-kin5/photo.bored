package Auth

import (
	"crypto/rand"
	"encoding/base64"
	"time"
)

const (
	// DefaultTokenDuration is the default expiration time for confirmation tokens
	DefaultTokenDuration = 5 * time.Minute
)

type ConfirmationToken struct {
	Token     string
	UserID    string
	ExpiresAt time.Time
}

// GenerateConfirmationToken creates a secure random token for email confirmation or 2-step verification
func GenerateConfirmationToken(userID string, duration time.Duration) (*ConfirmationToken, error) {
	if duration == 0 {
		duration = DefaultTokenDuration
	}
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	token := base64.URLEncoding.EncodeToString(b)
	return &ConfirmationToken{
		Token:     token,
		UserID:    userID,
		ExpiresAt: time.Now().Add(duration),
	}, nil
}

// ValidateConfirmationToken checks if the token is valid and not expired
func ValidateConfirmationToken(token *ConfirmationToken) bool {
	return token != nil && time.Now().Before(token.ExpiresAt)
}

package security

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"crypto/sha256"
	"encoding/hex"

	utils "app/pkg/utils"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenManager struct {
	config *Config
	db     *sql.DB
}

type TokenClaims struct {
	UserID    uuid.UUID `json:"user_id"`
	SessionID uuid.UUID `json:"session_id,omitempty"`
	TokenType string    `json:"token_type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type TokenValidationResult struct {
	Valid   bool
	Claims  *TokenClaims
	Error   error
	Expired bool
}

func NewTokenManager(config *Config, db *sql.DB) *TokenManager {
	return &TokenManager{
		config: config,
		db:     db,
	}
}

func (tm *TokenManager) GenerateTokenPair(userID uuid.UUID, sessionID uuid.UUID) (*Tokens, error) {
	now := time.Now()

	// Generate Access Token
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, TokenClaims{
		UserID:    userID,
		SessionID: sessionID,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(tm.config.AccessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "kitch-streaming",
			Subject:   userID.String(),
			ID:        sessionID.String(), // JWT ID for session tracking
		},
	})

	accessTokenString, err := accessToken.SignedString([]byte(tm.config.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Generate Refresh Token
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, TokenClaims{
		UserID:    userID,
		SessionID: sessionID,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(tm.config.RefreshTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "kitch-streaming",
			Subject:   userID.String(),
			ID:        sessionID.String(),
		},
	})

	refreshTokenString, err := refreshToken.SignedString([]byte(tm.config.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &Tokens{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    int64(tm.config.AccessTokenDuration.Seconds()),
	}, nil
}

func (tm *TokenManager) ValidateToken(tokenString string) (*TokenValidationResult, error) {
	// Check if token is blacklisted in database
	if tm.isTokenBlacklisted(tokenString) {
		return &TokenValidationResult{
			Valid: false,
			Error: errors.New("token has been revoked"),
		}, nil
	}

	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(tm.config.JWTSecret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return &TokenValidationResult{
				Valid:   false,
				Expired: true,
				Error:   errors.New("token has expired"),
			}, nil
		}
		return &TokenValidationResult{
			Valid: false,
			Error: fmt.Errorf("invalid token: %w", err),
		}, nil
	}

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		// Additional validation
		if claims.Issuer != "kitch-streaming" {
			return &TokenValidationResult{
				Valid: false,
				Error: errors.New("invalid token issuer"),
			}, nil
		}

		return &TokenValidationResult{
			Valid:  true,
			Claims: claims,
		}, nil
	}

	return &TokenValidationResult{
		Valid: false,
		Error: errors.New("invalid token claims"),
	}, nil
}

func (tm *TokenManager) RefreshToken(refreshTokenString string) (*Tokens, error) {
	result, err := tm.ValidateToken(refreshTokenString)
	if err != nil {
		return nil, err
	}

	if !result.Valid {
		return nil, result.Error
	}

	if result.Claims.TokenType != "refresh" {
		return nil, errors.New("invalid token type for refresh")
	}

	// Check if this refresh token has already been used (reuse detection)
	if tm.isRefreshTokenReused(refreshTokenString) {
		// This is a potential security breach - blacklist all tokens for this session
		tm.handleRefreshTokenReuse(result.Claims.UserID, result.Claims.SessionID)
		return nil, errors.New("refresh token reuse detected - security violation")
	}

	// Blacklist the current refresh token before issuing new ones
	err = tm.BlacklistToken(refreshTokenString, "refresh_rotation")
	if err != nil {
		return nil, fmt.Errorf("failed to blacklist old refresh token: %w", err)
	}

	// Generate new token pair with a new session ID for enhanced security
	newSessionID, err := GenerateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate new session ID: %w", err)
	}

	// Generate new token pair
	newTokens, err := tm.GenerateTokenPair(result.Claims.UserID, newSessionID)
	if err != nil {
		return nil, err
	}

	// Track the refresh token usage in the database
	err = tm.trackRefreshTokenUsage(result.Claims.UserID, newSessionID, newTokens.RefreshToken)
	if err != nil {
		// Log the error but don't fail the refresh operation
		utils.Logger.Errorf("Failed to track refresh token usage: %v", err)
	}

	return newTokens, nil
}

// isRefreshTokenReused checks if a refresh token has been used before
func (tm *TokenManager) isRefreshTokenReused(refreshTokenString string) bool {
	if tm.db == nil {
		return false // Skip check if no database
	}

	tokenHash := tm.hashToken(refreshTokenString)

	query := `
		SELECT EXISTS(
			SELECT 1 FROM token_blacklist 
			WHERE token_hash = $1 AND token_type = 'refresh'
		)
	`

	var exists bool
	err := tm.db.QueryRow(query, tokenHash).Scan(&exists)
	if err != nil {
		// Log error but don't fail validation - assume token is valid
		utils.Logger.Errorf("Failed to check refresh token reuse: %v", err)
		return false
	}

	return exists
}

// handleRefreshTokenReuse handles potential refresh token reuse attacks
func (tm *TokenManager) handleRefreshTokenReuse(userID uuid.UUID, sessionID uuid.UUID) {
	if tm.db == nil {
		return // Skip if no database
	}

	// Blacklist all tokens for this session
	query := `
		UPDATE user_sessions 
		SET is_active = false 
		WHERE user_id = $1 AND id = $2
	`

	_, err := tm.db.Exec(query, userID, sessionID)
	if err != nil {
		utils.Logger.Errorf("Failed to deactivate session after refresh token reuse: %v", err)
	}

	// Log security event
	utils.Logger.Warnf("Refresh token reuse detected for user %s, session %s - session deactivated", userID, sessionID)
}

// trackRefreshTokenUsage tracks refresh token usage for security monitoring
func (tm *TokenManager) trackRefreshTokenUsage(userID uuid.UUID, sessionID uuid.UUID, refreshToken string) error {
	if tm.db == nil {
		return nil // Skip if no database
	}

	tokenHash := tm.hashToken(refreshToken)

	query := `
		INSERT INTO user_sessions (user_id, id, token_hash, refresh_token_hash, expires_at, last_used)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (id) DO UPDATE SET
			refresh_token_hash = EXCLUDED.refresh_token_hash,
			last_used = NOW()
	`

	// Calculate expiration time (refresh token duration)
	expiresAt := time.Now().Add(tm.config.RefreshTokenDuration)

	_, err := tm.db.Exec(query, userID, sessionID, "", tokenHash, expiresAt)
	return err
}

// GetActiveSessionsCount returns the number of active sessions for a user
func (tm *TokenManager) GetActiveSessionsCount(userID uuid.UUID) (int, error) {
	if tm.db == nil {
		return 0, nil // Skip if no database
	}

	query := `
		SELECT COUNT(*) FROM user_sessions 
		WHERE user_id = $1 AND is_active = true AND expires_at > NOW()
	`

	var count int
	err := tm.db.QueryRow(query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get active sessions count: %w", err)
	}

	return count, nil
}

// RevokeAllUserSessions revokes all active sessions for a user (useful for security incidents)
func (tm *TokenManager) RevokeAllUserSessions(userID uuid.UUID, reason string) error {
	if tm.db == nil {
		return nil // Skip if no database
	}

	// Deactivate all sessions
	query := `
		UPDATE user_sessions 
		SET is_active = false 
		WHERE user_id = $1 AND is_active = true
	`

	result, err := tm.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke user sessions: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		utils.Logger.Infof("Revoked %d sessions for user %s: %s", rowsAffected, userID, reason)
	}

	return nil
}

func (tm *TokenManager) ExtractUserIDFromToken(tokenString string) (uuid.UUID, error) {
	result, err := tm.ValidateToken(tokenString)
	if err != nil {
		return uuid.Nil, err
	}

	if !result.Valid {
		return uuid.Nil, result.Error
	}

	return result.Claims.UserID, nil
}

// BlacklistToken adds a token to the database blacklist
func (tm *TokenManager) BlacklistToken(tokenString string, reason string) error {
	// Parse token to get claims for additional context
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(tm.config.JWTSecret), nil
	})

	if err != nil {
		return fmt.Errorf("failed to parse token for blacklisting: %w", err)
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok {
		return errors.New("invalid token claims for blacklisting")
	}

	// Hash the token for storage
	tokenHash := tm.hashToken(tokenString)

	// Insert into database
	query := `
		INSERT INTO token_blacklist (token_hash, user_id, session_id, token_type, expires_at, reason)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (token_hash) DO NOTHING
	`

	_, err = tm.db.Exec(query, tokenHash, claims.UserID, claims.SessionID, claims.TokenType, claims.ExpiresAt.Time, reason)
	if err != nil {
		return fmt.Errorf("failed to blacklist token: %w", err)
	}

	return nil
}

// isTokenBlacklisted checks if a token is in the database blacklist
func (tm *TokenManager) isTokenBlacklisted(tokenString string) bool {
	tokenHash := tm.hashToken(tokenString)

	query := `
		SELECT EXISTS(
			SELECT 1 FROM token_blacklist 
			WHERE token_hash = $1 AND expires_at > NOW()
		)
	`

	var exists bool
	err := tm.db.QueryRow(query, tokenHash).Scan(&exists)
	if err != nil {
		// Log error but don't fail validation - assume token is valid
		return false
	}

	return exists
}

// hashToken creates a SHA-256 hash of the token for secure storage
func (tm *TokenManager) hashToken(tokenString string) string {
	hash := sha256.Sum256([]byte(tokenString))
	return hex.EncodeToString(hash[:])
}

// CleanupExpiredBlacklistedTokens removes expired tokens from the blacklist
func (tm *TokenManager) CleanupExpiredBlacklistedTokens() error {
	query := `SELECT cleanup_expired_blacklisted_tokens()`

	var deletedCount int
	err := tm.db.QueryRow(query).Scan(&deletedCount)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired blacklisted tokens: %w", err)
	}

	if deletedCount > 0 {
		// Log cleanup for monitoring
		utils.Logger.Infof("Cleaned up %d expired blacklisted tokens", deletedCount)
	}

	return nil
}

// BlacklistUserSessions blacklists all active sessions for a user
func (tm *TokenManager) BlacklistUserSessions(userID uuid.UUID, reason string) error {
	query := `
		UPDATE user_sessions 
		SET is_active = false 
		WHERE user_id = $1 AND is_active = true
	`

	result, err := tm.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to blacklist user sessions: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		utils.Logger.Infof("Blacklisted %d sessions for user %s", rowsAffected, userID)
	}

	return nil
}

// Canonical GenerateSecureToken and GenerateSessionID live here for the security package
func GenerateSecureToken(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("token length must be positive")
	}

	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GenerateSessionID generates a new session ID
func GenerateSessionID() (uuid.UUID, error) {
	return uuid.NewRandom()
}

// ValidateTokenType checks if the token is of the expected type
func (tm *TokenManager) ValidateTokenType(tokenString, expectedType string) error {
	result, err := tm.ValidateToken(tokenString)
	if err != nil {
		return err
	}

	if !result.Valid {
		return result.Error
	}

	if result.Claims.TokenType != expectedType {
		return fmt.Errorf("expected token type %s, got %s", expectedType, result.Claims.TokenType)
	}

	return nil
}

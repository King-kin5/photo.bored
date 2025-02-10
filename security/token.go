package security

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

type TokenManager struct {
	config *Config
}

type TokenClaims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.StandardClaims
}

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func NewTokenManager(config *Config) *TokenManager {
	return &TokenManager{config: config}
}

func (tokenManager *TokenManager) GenerateTokenPair(UserID uuid.UUID)(*Tokens, error) {
	//Generate Access Token
	accessToken:=jwt.NewWithClaims(jwt.SigningMethodHS256,TokenClaims{
		UserID:UserID,
		StandardClaims:jwt.StandardClaims{
			ExpiresAt:time.Now().Add(tokenManager.config.AccessTokenDuration).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
	})
	accessTokenString, err := accessToken.SignedString([]byte(tokenManager.config.JWTSecret))
	if err != nil {
		return nil, err
	}
	// Generate Refresh Token
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, TokenClaims{
		UserID: UserID,
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: time.Now().Add(tokenManager.config.RefreshTokenDuration).Unix(),
				IssuedAt:  time.Now().Unix(),
			},
	})	
	refreshTokenString, err := refreshToken.SignedString([]byte(tokenManager.config.JWTSecret))
	if err != nil {
		return nil, err
	}
		
	return &Tokens{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
	}, nil
}
func (tm *TokenManager) ValidateToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(tm.config.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
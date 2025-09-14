package security

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

const (
	// Cookie names
	AccessTokenCookie  = "access_token"
	RefreshTokenCookie = "refresh_token"

	// Cookie settings
	CookieMaxAge   = 7 * 24 * 60 * 60 
	SecureCookie   = true             
	HttpOnlyCookie = true
	SameSiteCookie = http.SameSiteStrictMode
)

// CookieConfig holds cookie configuration
type CookieConfig struct {
	Domain   string
	Path     string
	Secure   bool
	HttpOnly bool
	SameSite http.SameSite
}

// DefaultCookieConfig returns production-ready cookie settings
func DefaultCookieConfig() *CookieConfig {
	return &CookieConfig{
		Domain:   "", // Empty for current domain
		Path:     "/",
		Secure:   SecureCookie,
		HttpOnly: HttpOnlyCookie,
		SameSite: SameSiteCookie,
	}
}

// SetAuthCookies sets HTTP-only cookies for access and refresh tokens
func SetAuthCookies(c echo.Context, accessToken, refreshToken string, config *CookieConfig) {
	if config == nil {
		config = DefaultCookieConfig()
	}

	// Set access token cookie (short-lived)
	accessCookie := &http.Cookie{
		Name:     AccessTokenCookie,
		Value:    accessToken,
		Path:     config.Path,
		Domain:   config.Domain,
		MaxAge:   int(time.Hour.Seconds()), // 1 hour
		Secure:   config.Secure,
		HttpOnly: config.HttpOnly,
		SameSite: config.SameSite,
	}
	c.SetCookie(accessCookie)

	// Set refresh token cookie (long-lived)
	refreshCookie := &http.Cookie{
		Name:     RefreshTokenCookie,
		Value:    refreshToken,
		Path:     config.Path,
		Domain:   config.Domain,
		MaxAge:   CookieMaxAge,
		Secure:   config.Secure,
		HttpOnly: config.HttpOnly,
		SameSite: config.SameSite,
	}
	c.SetCookie(refreshCookie)
}

// GetTokenFromCookie retrieves a token from cookies
func GetTokenFromCookie(c echo.Context, cookieName string) string {
	cookie, err := c.Cookie(cookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// GetAccessTokenFromCookie retrieves the access token from cookies
func GetAccessTokenFromCookie(c echo.Context) string {
	return GetTokenFromCookie(c, AccessTokenCookie)
}

// GetRefreshTokenFromCookie retrieves the refresh token from cookies
func GetRefreshTokenFromCookie(c echo.Context) string {
	return GetTokenFromCookie(c, RefreshTokenCookie)
}

// ClearAuthCookies removes authentication cookies
func ClearAuthCookies(c echo.Context, config *CookieConfig) {
	if config == nil {
		config = DefaultCookieConfig()
	}
	// Clear access token cookie
	accessCookie := &http.Cookie{
		Name:     AccessTokenCookie,
		Value:    "",
		Path:     config.Path,
		Domain:   config.Domain,
		MaxAge:   -1, // Delete immediately
		Secure:   config.Secure,
		HttpOnly: config.HttpOnly,
		SameSite: config.SameSite,
	}
	c.SetCookie(accessCookie)
	// Clear refresh token cookie
	refreshCookie := &http.Cookie{
		Name:     RefreshTokenCookie,
		Value:    "",
		Path:     config.Path,
		Domain:   config.Domain,
		MaxAge:   -1, // Delete immediately
		Secure:   config.Secure,
		HttpOnly: config.HttpOnly,
		SameSite: config.SameSite,
	}
	c.SetCookie(refreshCookie)
}

// RefreshAuthCookies updates the access token cookie with a new token
func RefreshAuthCookies(c echo.Context, accessToken string, config *CookieConfig) {
	if config == nil {
		config = DefaultCookieConfig()
	}
	// Update access token cookie
	accessCookie := &http.Cookie{
		Name:     AccessTokenCookie,
		Value:    accessToken,
		Path:     config.Path,
		Domain:   config.Domain,
		MaxAge:   int(time.Hour.Seconds()), // 1 hour
		Secure:   config.Secure,
		HttpOnly: config.HttpOnly,
		SameSite: config.SameSite,
	}
	c.SetCookie(accessCookie)
}

// IsSecureContext checks if the request is made over HTTPS
func IsSecureContext(c echo.Context) bool {
	// Check if the request is using HTTPS
	if c.Request().TLS != nil {
		return true
	}
	// Check for X-Forwarded-Proto header (common with reverse proxies)
	if c.Request().Header.Get("X-Forwarded-Proto") == "https" {
		return true
	}
	// Check for X-Forwarded-SSL header
	if c.Request().Header.Get("X-Forwarded-SSL") == "on" {
		return true
	}
	return false
}
// GetCookieConfigForContext returns appropriate cookie config based on context
func GetCookieConfigForContext(c echo.Context) *CookieConfig {
	config := DefaultCookieConfig()
	// In development or non-HTTPS environments, you might want to disable Secure flag
	if !IsSecureContext(c) {
		config.Secure = false
	}
	return config
}

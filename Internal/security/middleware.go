package security

import (
	utils "app/pkg/utils"
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// SecurityConfig holds security middleware configuration
type SecurityConfig struct {
	AllowedOrigins []string
	TrustedProxies []string
	EnableHSTS     bool
	EnableCSP      bool
	CSPDirectives  string
}

// DefaultSecurityConfig returns production-ready security settings
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		AllowedOrigins: []string{}, 
		TrustedProxies: []string{"127.0.0.1", "::1"},
		EnableHSTS:     true,
		EnableCSP:      true,
		CSPDirectives:  "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data: https:; font-src 'self' https:; connect-src 'self'; frame-ancestors 'none';",
	}
}

func SetupSecurityMiddleware(e *echo.Echo, config *Config, securityConfig *SecurityConfig) {
	if securityConfig == nil {
		securityConfig = DefaultSecurityConfig()
	}

	// Trust proxy middleware for proper IP detection
	// e.Use(middleware.ProxyWithConfig(middleware.ProxyConfig{
	// 	Balancer: middleware.NewRoundRobinBalancer(securityConfig.TrustedProxies),
	// }))

	// Use allowed origins from config if available, otherwise from security config
	allowedOrigins := config.AllowedOrigins
	if len(allowedOrigins) == 0 {
		allowedOrigins = securityConfig.AllowedOrigins
	}

	// CORS middleware
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "X-Requested-With"},
		ExposeHeaders:    []string{"X-Total-Count"},
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	}))

	// Security headers middleware
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY", // Changed from SAMEORIGIN for better security
		HSTSMaxAge:            31536000,
		HSTSExcludeSubdomains: false,
		ContentSecurityPolicy: securityConfig.CSPDirectives,
		ReferrerPolicy:        "strict-origin-when-cross-origin",
	}))

	// Request ID middleware for tracing
	e.Use(middleware.RequestIDWithConfig(middleware.RequestIDConfig{
		Generator: func() string {
			return uuid.New().String()
		},
	}))

	// Recover middleware
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		StackSize: 1 << 10, // 1 KB
		LogLevel:  1,       // Error level
	}))
}

// JWTMiddleware validates JWT tokens and sets user context
func JWTMiddleware(config *Config, db *sql.DB) echo.MiddlewareFunc {
	tokenManager := NewTokenManager(config, db)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing authorization header")
			}

			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid authorization header format")
			}

			result, err := tokenManager.ValidateToken(tokenParts[1])
			if err != nil {
				utils.Logger.Errorf("Token validation error: %v", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "Token validation failed")
			}

			if !result.Valid {
				if result.Expired {
					return echo.NewHTTPError(http.StatusUnauthorized, "Token has expired")
				}
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
			}

			// Set user context
			c.Set("user_id", result.Claims.UserID)
			c.Set("session_id", result.Claims.SessionID)
			c.Set("token_type", result.Claims.TokenType)

			return next(c)
		}
	}
}

// LoggingMiddleware logs request and response with structured logging
func LoggingMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()
		req := c.Request()
		res := c.Response()

		// Log request
		utils.Logger.Infof("Request started: %s %s from %s",
			req.Method, req.URL.String(), c.RealIP())

		err := next(c)
		duration := time.Since(start)

		// Log response
		status := res.Status
		if err != nil {
			if he, ok := err.(*echo.HTTPError); ok {
				status = he.Code
			} else {
				status = http.StatusInternalServerError
			}
		}

		utils.Logger.Infof("Request completed: %s %s - %d in %v",
			req.Method, req.URL.String(), status, duration)

		return err
	}
}


// AuthenticationMiddleware - FIXED version with correct excluded paths
func AuthenticationMiddleware(config *Config, db *sql.DB) echo.MiddlewareFunc {
	tokenManager := NewTokenManager(config, db)

	// Updated excluded paths to match your actual routes
	excludedPaths := []string{
		"/api/register",
		"/api/login",
		"/api/feed",        // Make feed data public
		"/health",
		"/static/",
		"/serveimage/",     // Make image serving public
		"/registration",
		"/login_page",
		"/feed",           // Template routes (HTML pages)
		
		"/ws/",            // WebSocket routes
		"/favicon.ico",
		"/robots.txt",
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()

			// Skip OPTIONS requests
			if req.Method == http.MethodOptions {
				return next(c)
			}

			// Skip excluded paths
			path := req.URL.Path
			for _, excludedPath := range excludedPaths {
				if strings.HasPrefix(path, excludedPath) {
					return next(c)
				}
			}

			utils.Logger.Debugf("Auth middleware checking path: %s", path)

			// FIXED: First try to get token from cookie (primary method)
			token := GetAccessTokenFromCookie(c)
			
			// Fallback to Authorization header for API clients (keep for compatibility)
			if token == "" {
				authHeader := req.Header.Get("Authorization")
				if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
					token = strings.TrimPrefix(authHeader, "Bearer ")
					utils.Logger.Debug("Using token from Authorization header")
				}
			} else {
				utils.Logger.Debug("Using token from cookie")
			}

			// If no token found anywhere, return unauthorized
			if token == "" {
				utils.Logger.Warn("No authentication token found")
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Authentication required",
				})
			}

			result, err := tokenManager.ValidateToken(token)
			if err != nil {
				utils.Logger.Errorf("Token validation error: %v", err)
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Token validation failed",
				})
			}

			if !result.Valid {
				if result.Expired {
					// Clear cookies on expired token
					cookieConfig := GetCookieConfigForContext(c)
					ClearAuthCookies(c, cookieConfig)
					
					return c.JSON(http.StatusUnauthorized, map[string]string{
						"error": "Token has expired",
						"code":  "TOKEN_EXPIRED",
					})
				}
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Invalid or expired token",
				})
			}

			// Validate token type
			if result.Claims.TokenType != "access" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Invalid token type",
				})
			}

			// Set user context
			c.Set("user_id", result.Claims.UserID)
			c.Set("session_id", result.Claims.SessionID)
			c.Set("user_claims", result.Claims)

			utils.Logger.Debugf("Authentication successful for user: %s", result.Claims.UserID)
			return next(c)
		}
	}
}

// TimeoutMiddleware adds request timeout
func TimeoutMiddleware(timeout time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx, cancel := context.WithTimeout(c.Request().Context(), timeout)
			defer cancel()

			c.SetRequest(c.Request().WithContext(ctx))

			done := make(chan error, 1)
			go func() {
				done <- next(c)
			}()

			select {
			case err := <-done:
				return err
			case <-ctx.Done():
				return echo.NewHTTPError(http.StatusRequestTimeout, "Request timeout")
			}
		}
	}
}

// AuditMiddleware logs security-relevant events
func AuditMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		path := req.URL.Path
		// Log authentication attempts
		if strings.Contains(req.URL.Path, "/auth/login") {
			utils.Logger.Infof("Login attempt from IP: %s, User-Agent: %s",
				c.RealIP(), req.UserAgent())
		}
				// Log registration attempts
		if strings.Contains(path, "/register") {
			utils.Logger.Infof("Registration attempt from IP: %s", c.RealIP())
		}
				// Log upload attempts
		if strings.Contains(path, "/upload") {
			userID := "anonymous"
			if uid, ok := c.Get("user_id").(uuid.UUID); ok {
				userID = uid.String()
			}
			utils.Logger.Infof("Upload attempt from user: %s, IP: %s", userID, c.RealIP())
		}

		err := next(c)

		// Log failed authentication
		if err != nil {
			if he, ok := err.(*echo.HTTPError); ok && he.Code == http.StatusUnauthorized {
				utils.Logger.Warnf("Failed authentication from IP: %s, Path: %s",
					c.RealIP(), req.URL.Path)
			}
		}

		return err
	}
}


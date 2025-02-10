package security
import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func SetupSecurityMiddleware(e *echo.Echo,config *Config){
	// CORS middleware
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: config.AllowedOrigins,
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	//Secure middleware

	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection: "1; mode=block",
		ContentTypeNosniff: "nosniff",
		XFrameOptions: "SAMEORIGIN",
		HSTSMaxAge: 31536000,
		HSTSExcludeSubdomains: false,
	}))

		// Rate limiting middleware
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))
}

func JWTMiddleware(Config *Config)echo.MiddlewareFunc{
	tokenManager:=NewTokenManager(Config)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader:=c.Request().Header.Get("Authorization")
			if authHeader ==""{
				return echo.NewHTTPError(http.StatusUnauthorized,"missing authorization Header")
			}

			tokenParts:=strings.Split(authHeader," ")
			if len(tokenParts)!=2||tokenParts[0]!="Bearer"{
				return echo.NewHTTPError(http.StatusUnauthorized,"invalid authrorization header format")
			}

			claims,err:=tokenManager.ValidateToken(tokenParts[1])
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}

			c.Set("user_id", claims.UserID)
			return next(c)
		}
	}
}

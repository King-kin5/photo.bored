// FIXED: Complete main.go integration
package main

import (
	"context"
	"database/sql"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/signal"

	"syscall"
	"time"

	database "app/Database"
	configs "app/configs"
	Auth "app/internal/Auth"
	"app/internal/security"
	"app/photo"
	utils "app/pkg/utils"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

type TemplateRenderer struct {
	templates *template.Template
}

func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {
	_ = godotenv.Load()
	utils.Init("info")
	utils.Logger.Info("Starting photo sharing server...")

	// Create Echo instance
	e := echo.New()

	// Load configuration first
	appConfig, err := configs.LoadConfig()
	if err != nil {
		utils.Logger.Fatalf("Failed to load configuration: %v", err)
	}

	if err := appConfig.Validate(); err != nil {
		utils.Logger.Fatalf("Configuration validation failed: %v", err)
	}

	// Create security config
	securityConfig := security.NewConfig()

	// Initialize security middleware BEFORE other middleware
	securityMiddlewareConfig := security.DefaultSecurityConfig()
	security.SetupSecurityMiddleware(e, securityConfig, securityMiddlewareConfig)

	// Add custom middleware in correct order
	e.Use(security.LoggingMiddleware)
	e.Use(security.AuditMiddleware)
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Add request logging middleware
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			utils.Logger.Debugf("Incoming request: %s %s", c.Request().Method, c.Request().URL.Path)
			return next(c)
		}
	})

	// Set up validator
	e.Validator = &CustomValidator{validator: validator.New()}

	// Set up template renderer
	renderer := &TemplateRenderer{
		templates: template.Must(template.ParseGlob("../templates/*.html")),
	}
	e.Renderer = renderer

	// Get the PostgreSQL database connection
	postgresDB, err := database.GetPostgresDB(appConfig)
	if err != nil {
		utils.Logger.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer postgresDB.Close()

	// Test database connection
	if err := postgresDB.Ping(); err != nil {
		utils.Logger.Fatalf("Database connection test failed: %v", err)
	}
	utils.Logger.Info("Database connection established")

	// Initialize WebSocket manager
	wsManager := photo.NewWebSocketManager()
	wsManager.StartBroadcaster()

	// Initialize the stores
	userStore := Auth.NewUserStore(postgresDB)
	photoStore := photo.NewPhotostore(postgresDB)

	// Initialize email service
	emailService, err := Auth.NewEmailService(appConfig)
	if err != nil {
		utils.Logger.Fatalf("Failed to initialize email service: %v", err)
	}

	// Initialize handlers
	userHandler := Auth.NewHandler(userStore, securityConfig, emailService, postgresDB)
	photoModel := photo.NewModel(photoStore, wsManager)

	// Setup routes in correct order
	setupPublicRoutes(e, userHandler, photoModel)
	setupProtectedRoutes(e, securityConfig, photoModel, postgresDB, userHandler)
	setupWebSocketRoutes(e, wsManager)

	// Print all registered routes for debugging
	utils.Logger.Info("=== Registered Routes ===")
	for _, route := range e.Routes() {
		utils.Logger.Infof("Route: %s %s", route.Method, route.Path)
	}
	utils.Logger.Info("========================")

	// Health check endpoint
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
			"version":   "1.0.0",
		})
	})

	// Setup graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start HTTP server in a goroutine
	go func() {
		utils.Logger.Info("HTTP server listening on :8080")
		if err := e.Start(":8080"); err != nil && err != http.ErrServerClosed {
			utils.Logger.Errorf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	utils.Logger.Info("Shutdown signal received, starting graceful shutdown...")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := e.Shutdown(shutdownCtx); err != nil {
		utils.Logger.Errorf("HTTP server shutdown error: %v", err)
	}

	utils.Logger.Info("Server shutdown complete")
}

// FIXED: setupPublicRoutes with public feed endpoint
func setupPublicRoutes(e *echo.Echo, userHandler *Auth.Handler, photoModel *photo.Model) {
	// Serve static files
	e.Static("/static", "../static")

	// Template routes
	e.GET("/registration", func(c echo.Context) error {
		return c.Render(http.StatusOK, "Registration.html", nil)
	})

	e.GET("/login_page", func(c echo.Context) error {
		return c.Render(http.StatusOK, "Login.html", nil)
	})

	e.GET("/feed", func(c echo.Context) error {
		return c.Render(http.StatusOK, "Feed.html", nil)
	})

	e.GET("/photo", func(c echo.Context) error {
		return c.Render(http.StatusOK, "photo.html", nil)
	})

	// FIXED: Public photo viewing route (no auth required)
	e.GET("/serveimage/:filename", photoModel.HandleServeImage)

	// FIXED: Public API endpoint for feed data
		// IMPROVED: Infinite scroll feed endpoint
	e.GET("/api/feed/infinite", photoModel.GetInfiniteFeed)
	
	// Keep the old endpoint for backward compatibility
	e.GET("/api/feed", func(c echo.Context) error {
		// Legacy pagination endpoint - redirect to infinite scroll
		return photoModel.GetInfiniteFeed(c)
	})

	// Authentication API routes (public)
	e.POST("/api/register", userHandler.RegisterUser)
	e.POST("/api/login", userHandler.LoginUser)

	utils.Logger.Info("Public routes configured")
}

// FIXED: setupProtectedRoutes with proper authentication
func setupProtectedRoutes(e *echo.Echo, config *security.Config, photoModel *photo.Model, db *sql.DB, userHandler *Auth.Handler) {
	// Create the protected group with authentication middleware
	api := e.Group("/api")
	
	// Add debug middleware for protected routes
	api.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			utils.Logger.Debugf("Protected route accessed: %s %s", c.Request().Method, c.Request().URL.Path)
			return next(c)
		}
	})
	
	// Apply authentication middleware to all protected routes
	api.Use(security.AuthenticationMiddleware(config, db))

	// User profile routes
	api.GET("/user/profile/:id", userHandler.Profile)
	api.GET("/user/profile", func(c echo.Context) error {
		userID, ok := c.Get("user_id").(uuid.UUID)
		if !ok {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "User not authenticated",
			})
		}
		c.SetParamNames("id")
		c.SetParamValues(userID.String())
		return userHandler.Profile(c)
	})
	api.PUT("/user/profile/:id", userHandler.UpdateUser)
	api.GET("/user/username/:username", userHandler.GetProfileByUsername)
	
	// Authentication routes that require being logged in
	api.POST("/logout", userHandler.LogoutUser)
	api.POST("/refresh", userHandler.RefreshToken)

	// FIXED: Photo upload endpoint with enhanced logging
	api.POST("/upload", func(c echo.Context) error {
		utils.Logger.Info("Upload endpoint accessed")
		
		// Additional validation
		userID, ok := c.Get("user_id").(uuid.UUID)
		if !ok {
			utils.Logger.Error("Upload attempted without valid user context")
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "User authentication required for upload",
			})
		}
		
		utils.Logger.Infof("Processing upload for user: %s", userID)
		return photoModel.HandleUpload(c)
	})

	// Other protected photo operations
	api.DELETE("/photos/delete/:filename", photoModel.HandleDeleteImage)
	api.POST("/photos/caption/:filename", photoModel.HandleCaption)
	api.GET("/photos/:photoId/comments", photoModel.GetPhotoComments)

	// Protected album operations
	albums := api.Group("/albums")
	albums.POST("", photoModel.HandleCreateAlbum)
	albums.POST("/photo", photoModel.HandleAddPhotoToAlbum)

	utils.Logger.Info("Protected routes configured")
}

// setupWebSocketRoutes configures WebSocket routes
func setupWebSocketRoutes(e *echo.Echo, wsManager *photo.WebSocketManager) {
	e.GET("/ws/feed", wsManager.HandleWebSocket)
	utils.Logger.Info("WebSocket routes configured")
}
package main

import (
	"context"
	"database/sql"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	database "app/Database"
	configs "app/configs"
	Auth "app/internal/Auth"
	"app/internal/security"
	"app/photo"
	utils "app/pkg/utils"

	"github.com/go-playground/validator/v10"
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

	// Add request logging middleware
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			utils.Logger.Infof("Incoming request: %s %s", c.Request().Method, c.Request().URL.Path)
			return next(c)
		}
	})

	// Load configuration
	appConfig, err := configs.LoadConfig()
	if err != nil {
		utils.Logger.Fatalf("Failed to load configuration: %v", err)
	}

	if err := appConfig.Validate(); err != nil {
		utils.Logger.Fatalf("Configuration validation failed: %v", err)
	}

	// Create security config
	securityConfig := security.NewConfig()

	// Initialize security middleware
	securityMiddlewareConfig := security.DefaultSecurityConfig()
	security.SetupSecurityMiddleware(e, securityConfig, securityMiddlewareConfig)

	// Add custom middleware
	e.Use(security.LoggingMiddleware)
	e.Use(security.AuditMiddleware)
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Set up validator
	e.Validator = &CustomValidator{validator: validator.New()}

	// Set up template renderer
	renderer := &TemplateRenderer{
		templates: template.Must(template.ParseGlob("templates/*.html")),
	}
	e.Renderer = renderer

	// Get the PostgreSQL database connection
	postgresDB, err := database.GetPostgresDB(appConfig)
	if err != nil {
		utils.Logger.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer postgresDB.Close()

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

	// Setup routes
	setupPublicRoutes(e, userHandler, photoModel)
	setupProtectedRoutes(e, securityConfig, photoModel, postgresDB)
	setupWebSocketRoutes(e, wsManager)

	// Print all registered routes for debugging
	for _, route := range e.Routes() {
		utils.Logger.Infof("Registered route: %s %s", route.Method, route.Path)
	}

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

// setupPublicRoutes configures public routes that don't require authentication
func setupPublicRoutes(e *echo.Echo, userHandler *Auth.Handler, photoModel *photo.Model) {
	// Serve static files
	e.Static("/static", "static")

	// Template routes
	e.GET("/registration", func(c echo.Context) error {
		return c.Render(http.StatusOK, "Registration.html", nil)
	})

	e.GET("/login_page", func(c echo.Context) error {
		return c.Render(http.StatusOK, "Login.html", nil)
	})

	// Public photo viewing route
	e.GET("/serveimage/:filename", photoModel.HandleServeImage)

	// Public feed endpoint
	e.GET("/feed", func(c echo.Context) error {
		pageStr := c.QueryParam("page")
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}

		const itemsPerPage = 100
		offset := (page - 1) * itemsPerPage

		photos, err := photoModel.GetPaginatedPhotos(itemsPerPage, offset)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to load photos",
			})
		}

		// Ensure photos have valid dates
		for i := range photos {
			if photos[i].Date.IsZero() {
				photos[i].Date = time.Now()
			}
		}

		return c.JSON(http.StatusOK, photos)
	})

	// Authentication API routes
	auth := e.Group("/api/v1/auth")
	auth.POST("/register", userHandler.RegisterUser)
	auth.POST("/login", userHandler.LoginUser)
	auth.POST("/refresh-token", userHandler.RefreshToken)
	auth.POST("/logout", userHandler.LogoutUser)

	utils.Logger.Info("Public routes configured")
}

// setupProtectedRoutes configures routes that require authentication
func setupProtectedRoutes(e *echo.Echo, config *security.Config, photoModel *photo.Model, db *sql.DB) {
	// Create the protected group with authentication middleware
	api := e.Group("/api/v1")
	api.Use(security.AuthenticationMiddleware(config, db))

	// Protected photo template route
	api.GET("/photo", func(c echo.Context) error {
		return c.Render(http.StatusOK, "photo.html", nil)
	})

	// Protected photo operations
	photos := api.Group("/photos")
	photos.POST("/upload", photoModel.HandleUpload)
	photos.DELETE("/delete/:filename", photoModel.HandleDeleteImage)
	photos.POST("/caption/:filename", photoModel.HandleCaption)
	photos.GET("/:photoId/comments", photoModel.GetPhotoComments)

	// Protected album operations
	albums := api.Group("/albums")
	albums.POST("", photoModel.HandleCreateAlbum)
	albums.POST("/photo", photoModel.HandleAddPhotoToAlbum)

	utils.Logger.Info("Protected routes configured")
}

// setupWebSocketRoutes configures WebSocket routes
func setupWebSocketRoutes(e *echo.Echo, wsManager *photo.WebSocketManager) {
	e.GET("/ws", wsManager.HandleWebSocket)
	utils.Logger.Info("WebSocket routes configured")
}
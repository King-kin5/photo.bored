package main

import (
    "app/photo"
    "app/security"
    "app/user"
    
    "html/template"
    "io"
    "log"
    "net/http"
    "strconv"
    "time"

    "github.com/go-playground/validator/v10"
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
    // Create a new Echo instance
    e := echo.New()

    // Set up middleware
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
    postgresDB, err := user.GetPostgresDB()
    if err != nil {
        log.Fatalf("Failed to connect to PostgreSQL: %v", err)
    }
    defer postgresDB.Close()
    //Initialize Wbesocket
    wsManager := photo.NewWebSocketManager()
    wsManager.StartBroadcaster()
    // Initialize the stores
    userstore := user.NewUserStore(postgresDB)
    photostore := photo.NewPhotostore(postgresDB)

    // Setup security config
    securityConfig := security.NewConfig()

    // Setup security middleware
    security.SetupSecurityMiddleware(e, securityConfig)

    // Initialize handlers
    userHandler := user.NewHandler(*userstore, securityConfig)
    photoModel := photo.NewModel(photostore, wsManager)


    // WebSocket route
    e.GET("/ws", wsManager.HandleWebSocket)

    // Serve static files
    e.Static("/static", "static")

    // Public routes
    e.GET("/registration", func(c echo.Context) error {
        return c.Render(http.StatusOK, "Registration.html", nil)
    })

    e.GET("/login_page", func(c echo.Context) error {
        return c.Render(http.StatusOK, "Login.html", nil)
    })

    e.GET("/feed", func(c echo.Context) error {
        return c.Render(http.StatusOK, "Feed.html", nil)
    })

    // Authentication routes
    e.POST("/api/register", userHandler.RegisterUser)
    e.POST("/api/login", userHandler.LoginUser)

    // Public photo viewing route
    e.GET("/serveimage/:filename", photoModel.HandleServeImage)

    // Protected routes group
    api := e.Group("/api")
    api.Use(security.JWTMiddleware(securityConfig))

    // Protected photo routes
    api.GET("/photo", func(c echo.Context) error {
        return c.Render(http.StatusOK, "photo.html", nil)
    })

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

        for i := range photos {
            if photos[i].Date.IsZero() {
                photos[i].Date = time.Now()
            }
        }

        return c.JSON(http.StatusOK, photos)
    })

    // Protected photo operations
    api.POST("/upload", photoModel.HandleUpload)
    api.DELETE("/delete/:filename", photoModel.HandleDeleteImage)
    api.POST("/caption/:filename", photoModel.HandleCaption)

    // Protected album operations
    api.POST("/album", photoModel.HandleCreateAlbum)
    api.POST("/album/photo", photoModel.HandleAddPhotoToAlbum)
    api.GET("/photo/:photoId/comments", photoModel.GetPhotoComments)

    // Start server
    log.Println("Server is running on port 8080")
    e.Logger.Fatal(e.Start(":8080"))
}
package user

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
    "app/security"
)

type User struct {
	UserID     uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Password string    `json:"password"`
}

// UserInput represents the incoming registration/login data
type UserInput struct {
	Username string `json:"username" form:"username"`
	Email    string `json:"email" form:"email"`
	Password string `json:"password" form:"password"`
}


type Handler struct {
	userstore    UserStore
	tokenManager *security.TokenManager
}

func NewHandler(userstore UserStore, config *security.Config) *Handler {
	return &Handler{
		userstore:    userstore,
		tokenManager: security.NewTokenManager(config),
	}
}


// RegisterUser handles user registration
func (h *Handler) RegisterUser(c echo.Context) error {
    var input UserInput

    // Debug log before binding
    log.Printf("Received registration request")

    // Bind the request data
    if err := c.Bind(&input); err != nil {
        log.Printf("Error binding request data: %v", err)
        return c.String(http.StatusBadRequest, "Invalid input")
    }

    // Debug log after binding
    log.Printf("Received input: %+v", input)

    // Validate input
    if input.Username == "" || input.Email == "" || input.Password == "" {
        log.Printf("Validation failed: missing required fields")
        return c.String(http.StatusBadRequest, "All fields are required")
    }

    // Check the database connection
    if err := h.userstore.CheckDBConnection(); err != nil {
        log.Printf("Database connection error: %v", err)
        return c.String(http.StatusInternalServerError, "Database connection error")
    }

    // Check if email exists
    existingUser, err := h.userstore.GetUserByEmail(input.Email)
    if err != nil {
        log.Printf("Error checking existing email: %v", err)
        return c.String(http.StatusInternalServerError, "Failed to check if email exists")
    }
    if existingUser != nil {
        log.Printf("Email already exists: %s", input.Email)
        return c.String(http.StatusConflict, "Email already exists")
    }

    // Hash password
    hashedPassword, err := PasswordHash(input.Password)
    if err != nil {
        log.Printf("Error hashing password: %v", err)
        return c.String(http.StatusInternalServerError, "Failed to hash password")
    }

    // Create user
    user := &User{
        UserID:   uuid.New(),
        Username: input.Username,
        Email:    input.Email,
        Password: hashedPassword,
    }

    if err := h.userstore.CreateUser(user); err != nil {
        log.Printf("Error creating user: %v", err)
        return c.String(http.StatusInternalServerError, "Failed to register user")
    }

    // Generate token pair for automatic login
    tokens, err := h.tokenManager.GenerateTokenPair(user.UserID)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate tokens")
    }

    log.Printf("User registered and logged in successfully: %s", input.Email)
	log.Printf("Acess  token : %s", tokens)
    return c.JSON(http.StatusCreated, map[string]interface{}{
        "message": "User registered successfully",
        "tokens":  tokens,
    })
}
// LoginUser handles user login
func (h *Handler) LoginUser(c echo.Context) error {
	var input UserInput

	// Bind the request data (handles both JSON and form data)
	if err := c.Bind(&input); err != nil {
		log.Printf("Error binding request data: %v", err)
		return c.String(http.StatusBadRequest, "Invalid input")
	}

	log.Printf("Received login input: %+v", input)

	// Validate input
	if input.Email == "" || input.Password == "" {
		log.Printf("Login failed: Email or Password is empty")
		return c.String(http.StatusBadRequest, "Email and Password are required")
	}

	// Get user by email
	user, err := h.userstore.GetUserByEmail(input.Email)
	if err != nil {
		log.Printf("UserLogin: Failed to get user by email: %v\n", err)
		return c.String(http.StatusInternalServerError, "Login failed: Internal server error")
	}

	// Check if user exists
	if user == nil {
		log.Printf("Login failed: Invalid email or password")
		return c.String(http.StatusUnauthorized, "Invalid email or password")
	}

	// Validate password
	if !CheckPasswordSame(user.Password, input.Password) {
		log.Printf("Login failed: Invalid email or password")
		return c.String(http.StatusUnauthorized, "Invalid email or password")
	}

	// Generate token pair
	tokens, err := h.tokenManager.GenerateTokenPair(user.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate tokens")
	}

	log.Printf("User login successful for email: %s", input.Email)
	log.Printf("Acess  token : %s", tokens)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "User login successful",
		"tokens":  tokens,
	})
}

func (h *Handler) RefreshToken(c echo.Context) error {
	refreshToken := c.FormValue("refresh_token")
	if refreshToken == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Refresh token is required")
	}
	
	claims, err := h.tokenManager.ValidateToken(refreshToken)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid refresh token")
	}
	
    // Generate new token pair
	tokens, err := h.tokenManager.GenerateTokenPair(claims.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate tokens")
	}
	
	return c.JSON(http.StatusOK, tokens)
	
}
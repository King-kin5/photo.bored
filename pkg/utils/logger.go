package utils
import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}
func (e *AppError) Error() string {
	return e.Message
}
// Predefined errors
var (
	ErrInvalidCredentials = &AppError{
		Code:    http.StatusUnauthorized,
		Message: "Invalid credentials",
	}
	ErrUserNotFound = &AppError{
		Code:    http.StatusNotFound,
		Message: "User not found",
	}
	ErrUserAlreadyExists = &AppError{
		Code:    http.StatusConflict,
		Message: "User already exists",
	}
	ErrInvalidToken = &AppError{
		Code:    http.StatusUnauthorized,
		Message: "Invalid or expired token",
	}
	ErrAccessDenied = &AppError{
		Code:    http.StatusForbidden,
		Message: "Access denied",
	}
	ErrValidationFailed = &AppError{
		Code:    http.StatusBadRequest,
		Message: "Validation failed",
	}
	ErrStreamKeyNotFound = &AppError{
		Code:    http.StatusNotFound,
		Message: "Stream key not found",
	}
	ErrInternalServer = &AppError{
		Code:    http.StatusInternalServerError,
		Message: "Internal server error",
	}
)

func NewAppError(code int, message string, details ...string) *AppError {
	err := &AppError{
		Code:    code,
		Message: message,
	}
	if len(details) > 0 {
		err.Details = details[0]
	}
	return err
}

func NewValidationError(message string) *AppError {
	return &AppError{
		Code:    http.StatusBadRequest,
		Message: message,
	}
}

func NewInternalError(message string) *AppError {
	return &AppError{
		Code:    http.StatusInternalServerError,
		Message: message,
	}
}

// CustomHTTPErrorHandler handles errors across the application
func CustomHTTPErrorHandler(err error, c echo.Context) {
	var appErr *AppError

	// Check if it's our custom AppError
	if ae, ok := err.(*AppError); ok {
		appErr = ae
	} else if he, ok := err.(*echo.HTTPError); ok {
		// Handle echo HTTP errors
		appErr = &AppError{
			Code:    he.Code,
			Message: fmt.Sprintf("%s", he.Message),
		}
	} else {
		// Handle generic errors
		appErr = &AppError{
			Code:    http.StatusInternalServerError,
			Message: "Internal server error",
			Details: err.Error(),
		}
	}
	// Log the error
	WithFields(map[string]interface{}{
		"error":  err.Error(),
		"code":   appErr.Code,
		"path":   c.Request().URL.Path,
		"method": c.Request().Method,
	}).Error("HTTP Error")

	// Don't expose internal error details in production
	if appErr.Code == http.StatusInternalServerError {
		appErr.Details = ""
	}
	// Send error response
	if !c.Response().Committed {
		if c.Request().Header.Get("Content-Type") == "application/json" {
			c.JSON(appErr.Code, appErr)
		} else {
			c.JSON(appErr.Code, appErr)
		}
	}
}

// SplitString splits a comma-separated string into a slice, trimming whitespace
func SplitString(input string) []string {
	if input == "" {
		return []string{}
	}
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

package security
import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

// PasswordHash hashes a password using bcrypt with a cost of 12
func PasswordHash(password string) (string, error) {
	if len(password) == 0 {
		return "", fmt.Errorf("password cannot be empty")
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedPassword), nil
}

// CheckPasswordSame compares a hashed password with a plain text password
func CheckPasswordSame(hashedPassword, password string) bool {
	if len(hashedPassword) == 0 || len(password) == 0 {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// GenerateRandomcode generates a random numeric code of the specified length
func GenerateRandomcode(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("code length must be positive")
	}
	code := ""
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("failed to generate random code: %w", err)
		}
		code += strconv.Itoa(int(num.Int64()))
	}
	return code, nil
}
// ValidatePasswordStrength validates password strength requirements
func ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}
	if len(password) > 128 {
		return fmt.Errorf("password must be less than 128 characters")
	}
	var (
		hasUpper   = false
		hasLower   = false
		hasDigit   = false
		hasSpecial = false
	)
	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		case isSpecialChar(char):
			hasSpecial = true
		}
	}
	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !hasDigit {
		return fmt.Errorf("password must contain at least one digit")
	}
	if !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}
	return nil
}

// isSpecialChar checks if a character is a special character
func isSpecialChar(char rune) bool {
	specialChars := "!@#$%^&*(),.?\":{}|<>"
	for _, special := range specialChars {
		if char == special {
			return true
		}
	}
	return false
}